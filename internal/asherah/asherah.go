package asherah

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/godaddy/asherah-cobhan/internal/log"
	"github.com/godaddy/asherah/go/appencryption"
	"github.com/godaddy/asherah/go/appencryption/pkg/crypto/aead"
	"github.com/godaddy/asherah/go/appencryption/pkg/kms"
	"github.com/godaddy/asherah/go/appencryption/pkg/persistence"
	"github.com/godaddy/asherah/go/securememory/memguard"
)

var globalSessionFactory *appencryption.SessionFactory
var globalInitialized int32 = 0

var ErrAsherahAlreadyInitialized = errors.New("asherah already initialized")
var ErrAsherahNotInitialized = errors.New("asherah not initialized")
var ErrAsherahFailedInitialization = errors.New("asherah failed initialization")

func Setup(options *Options) error {
	if atomic.LoadInt32(&globalInitialized) == 1 {
		output.StderrDebugLog("Failed to initialize asherah: already initialized")
		return ErrAsherahAlreadyInitialized
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
			Policy:  NewCryptoPolicy(options),
		},
		NewMetastore(options),
		NewKMS(options, crypto),
		crypto,
		appencryption.WithSecretFactory(new(memguard.SecretFactory)),
		appencryption.WithMetrics(false),
	)

	if globalSessionFactory == nil {
		output.StderrDebugLog("Failed to create session factory")
		return ErrAsherahFailedInitialization
	}

	atomic.StoreInt32(&globalInitialized, 1)
	return nil
}

func Shutdown() {
	if atomic.CompareAndSwapInt32(&globalInitialized, 1, 0) {
		globalSessionFactory.Close()
		globalSessionFactory = nil
	}
}

func Encrypt(partitionId string, data []byte) (*appencryption.DataRowRecord, error) {
	if globalInitialized == 0 {
		output.StderrDebugLog("Failed to encrypt data: asherah is not initialized")
		return nil, ErrAsherahNotInitialized
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		output.StderrDebugLogf("Failed to get session for partition %v: %v", partitionId, err.Error())
		return nil, err
	}
	defer session.Close()

	ctx := context.Background()
	return session.Encrypt(ctx, data)
}

func Decrypt(partitionId string, drr *appencryption.DataRowRecord) ([]byte, error) {
	if globalInitialized == 0 {
		return nil, ErrAsherahNotInitialized
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		output.StderrDebugLogf("Failed to get session for partition %v: %v", partitionId, err.Error())
		return nil, err
	}
	defer session.Close()

	ctx := context.Background()
	return session.Decrypt(ctx, *drr)
}

func NewMetastore(opts *Options) appencryption.Metastore {
	switch opts.Metastore {
	case "rdbms":
		// TODO: support other databases
		db, err := newMysql(opts.ConnectionString)
		if err != nil {
			output.StderrDebugLogf("PANIC: Failed to connect to database: %v", err.Error())
			panic(err)
		}

		// set optional replica read consistency
		if len(opts.ReplicaReadConsistency) > 0 {
			err := setRdbmsReplicaReadConsistencyValue(opts.ReplicaReadConsistency)
			if err != nil {
				output.StderrDebugLogf("PANIC: Failed to set replica read consistency: %v", err.Error())
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
	case "test-debug-memory":
		// We don't warn if the user specifically asks for test-debug-memory
		return persistence.NewMemoryMetastore()
	case "memory":
		output.StderrDebugLog("*** WARNING WARNING WARNING USING MEMORY METASTORE - THIS IS FOR TEST/DEBUG ONLY ***")
		return persistence.NewMemoryMetastore()
	default:
		output.StderrDebugLogf("PANIC: Unknown metastore type: %v", opts.Metastore)
		panic("Unknown metastore type")
	}
}

func NewKMS(opts *Options, crypto appencryption.AEAD) appencryption.KeyManagementService {
	if opts.KMS == "static" {
		output.StderrDebugLog("*** WARNING WARNING WARNING USING STATIC MASTER KEY - THIS IS FOR TEST/DEBUG ONLY ***")

		m, err := kms.NewStatic("thisIsAStaticMasterKeyForTesting", aead.NewAES256GCM())
		if err != nil {
			output.StderrDebugLogf("PANIC: Failed to create static master key: %v", err.Error())
			panic(err)
		}

		return m
	} else if opts.KMS == "test-debug-static" {
		// We don't warn if the user specifically asks for test-debug-static
		m, err := kms.NewStatic("thisIsAStaticMasterKeyForTesting", crypto)
		if err != nil {
			output.StderrDebugLogf("PANIC: Failed to create static master key: %v", err.Error())
			panic(err)
		}

		return m
	}

	m, err := kms.NewAWS(crypto, opts.PreferredRegion, opts.RegionMap)
	if err != nil {
		output.StderrDebugLogf("PANIC: Failed to create AWS KMS: %v", err.Error())
		panic(err)
	}

	return m
}

func NewCryptoPolicy(options *Options) *appencryption.CryptoPolicy {
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
