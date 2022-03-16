package asherah_internals

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/godaddy/asherah/go/appencryption"
	"github.com/godaddy/asherah/go/appencryption/pkg/crypto/aead"
	"github.com/godaddy/asherah/go/appencryption/pkg/kms"
	"github.com/godaddy/asherah/go/appencryption/pkg/persistence"
	"github.com/godaddy/asherah/go/securememory/memguard"
)

var globalSessionFactory *appencryption.SessionFactory
var globalInitialized int32 = 0

func SetupAsherah(options *Options) error {
	if globalInitialized != 0 {
		return errors.New("Asherah is already initialized")
	}

	crypto := aead.NewAES256GCM()

	if options.SessionCacheMaxSize == 0 {
		options.SessionCacheMaxSize = appencryption.DefaultSessionCacheMaxSize
	}

	if options.SessionCacheDuration == 0 {
		options.SessionCacheDuration = appencryption.DefaultSessionCacheDuration
	}

	if options.ExpireAfter == 0 {
		options.ExpireAfter = appencryption.DefaultExpireAfter
	}

	if options.CheckInterval == 0 {
		options.CheckInterval = appencryption.DefaultRevokedCheckInterval
	}

	globalSessionFactory = appencryption.NewSessionFactory(
		&appencryption.Config{
			Service: options.ServiceName,
			Product: options.ProductID,
			Policy:  newCryptoPolicy(options),
		},
		newMetastore(options),
		newKMS(options, crypto),
		crypto,
		appencryption.WithSecretFactory(new(memguard.SecretFactory)),
		appencryption.WithMetrics(false),
	)
	return nil
}

func ShutdownAsherah() {
	if globalInitialized != 0 {
		globalSessionFactory.Close()
		globalSessionFactory = nil
		atomic.StoreInt32(&globalInitialized, 0)
	}
}

func Encrypt(partitionId string, payload []byte) (*appencryption.DataRowRecord, error) {
	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	ctx := context.Background()
	return session.Encrypt(ctx, payload)
}

func Decrypt(partitionId string, drr *appencryption.DataRowRecord) ([]byte, error) {
	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	ctx := context.Background()
	return session.Decrypt(ctx, *drr)
}

func newCryptoPolicy(options *Options) *appencryption.CryptoPolicy {
	policyOpts := []appencryption.PolicyOption{
		appencryption.WithExpireAfterDuration(options.ExpireAfter),
		appencryption.WithRevokeCheckInterval(options.CheckInterval),
	}

	if options.EnableSessionCaching {
		policyOpts = append(policyOpts,
			appencryption.WithSessionCache(),
			appencryption.WithSessionCacheMaxSize(options.SessionCacheMaxSize),
			appencryption.WithSessionCacheDuration(options.SessionCacheDuration),
		)
	}

	return appencryption.NewCryptoPolicy(policyOpts...)
}

func newMetastore(opts *Options) appencryption.Metastore {
	switch opts.Metastore {
	case "rdbms":
		// TODO: support other databases
		db, err := NewMysql(opts.ConnectionString)
		if err != nil {
			panic(err)
		}

		// set optional replica read consistency
		if len(opts.ReplicaReadConsistency) > 0 {
			err := SetRdbmsReplicaReadConsistencyValue(opts.ReplicaReadConsistency)
			if err != nil {
				panic(err)
			}
		}

		return persistence.NewSQLMetastore(db)
	case "dynamodb":
		awsOpts := awssession.Options{
			SharedConfigState: awssession.SharedConfigEnable,
		}

		if len(opts.DynamoDBEndpoint) > 0 {
			awsOpts.Config.Endpoint = aws.String(opts.DynamoDBEndpoint)
		}

		if len(opts.DynamoDBRegion) > 0 {
			awsOpts.Config.Region = aws.String(opts.DynamoDBRegion)
		}

		return persistence.NewDynamoDBMetastore(
			awssession.Must(awssession.NewSessionWithOptions(awsOpts)),
			persistence.WithDynamoDBRegionSuffix(opts.EnableRegionSuffix),
			persistence.WithTableName(opts.DynamoDBTableName),
		)
	default:
		return persistence.NewMemoryMetastore()
	}
}

func newKMS(opts *Options, crypto appencryption.AEAD) appencryption.KeyManagementService {
	if opts.KMS == "static" {
		m, err := kms.NewStatic("thisIsAStaticMasterKeyForTesting", aead.NewAES256GCM())
		if err != nil {
			panic(err)
		}

		return m
	}

	m, err := kms.NewAWS(crypto, opts.PreferredRegion, opts.RegionMap)
	if err != nil {
		panic(err)
	}

	return m
}
