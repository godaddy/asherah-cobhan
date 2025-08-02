package asherah

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	log "github.com/godaddy/asherah-cobhan/internal/log"
	"github.com/godaddy/asherah/go/appencryption"
	"github.com/godaddy/asherah/go/appencryption/pkg/crypto/aead"
	"github.com/godaddy/asherah/go/appencryption/pkg/kms"
	asherahLog "github.com/godaddy/asherah/go/appencryption/pkg/log"
	"github.com/godaddy/asherah/go/appencryption/pkg/persistence"
	"github.com/godaddy/asherah/go/securememory/memguard"
)

var globalSessionFactory *appencryption.SessionFactory
var globalInitialized int32 = 0

var ErrAsherahAlreadyInitialized = errors.New("asherah already initialized")
var ErrAsherahNotInitialized = errors.New("asherah not initialized")
var ErrAsherahFailedInitialization = errors.New("asherah failed initialization")

type logFunc func(format string, v ...interface{})

func (f logFunc) Debugf(format string, v ...interface{}) {
	f(format, v...)
}

func Setup(options *Options) error {
	if atomic.LoadInt32(&globalInitialized) == 1 {
		log.ErrorLog("Failed to initialize asherah: already initialized")
		return ErrAsherahAlreadyInitialized
	}

	if options.Verbose && log.DebugLogf != nil {
		asherahLog.SetLogger(logFunc(log.DebugLogf))
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
		log.ErrorLog("Failed to create session factory")
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
		log.ErrorLog("Failed to encrypt data: asherah is not initialized")
		return nil, ErrAsherahNotInitialized
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		log.ErrorLogf("Failed to get session for partition %v: %v", partitionId, err.Error())
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
		log.ErrorLogf("Failed to get session for partition %v: %v", partitionId, err.Error())
		return nil, err
	}
	defer session.Close()

	ctx := context.Background()
	return session.Decrypt(ctx, *drr)
}

func NewMetastore(opts *Options) appencryption.Metastore {
	switch opts.Metastore {
	case "rdbms":
		var dbType string
		if len(opts.SQLMetastoreDBType) > 1 {
			dbType = opts.SQLMetastoreDBType
		} else {
			dbType = "mysql"
		}
		db, err := newConnection(dbType, opts.ConnectionString)
		if err != nil {
			log.ErrorLogf("PANIC: Failed to connect to %s database with connection string: %v", dbType, err.Error())
			panic(fmt.Errorf("failed to connect to %s database: %w", dbType, err))
		}

		// set optional replica read consistency
		if len(opts.ReplicaReadConsistency) > 0 {
			err := setRdbmsReplicaReadConsistencyValue(opts.ReplicaReadConsistency)
			if err != nil {
				log.ErrorLogf("PANIC: Failed to set replica read consistency to '%s': %v", opts.ReplicaReadConsistency, err.Error())
				panic(fmt.Errorf("failed to set replica read consistency to '%s': %w", opts.ReplicaReadConsistency, err))
			}
		}

		return persistence.NewSQLMetastore(db, persistence.WithSQLMetastoreDBType(persistence.SQLMetastoreDBType(dbType)))
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
		log.ErrorLog("*** WARNING WARNING WARNING USING MEMORY METASTORE - THIS IS FOR TEST/DEBUG ONLY ***")
		return persistence.NewMemoryMetastore()
	default:
		log.ErrorLogf("PANIC: Unknown metastore type: %v (valid options: rdbms, dynamodb, memory)", opts.Metastore)
		panic(fmt.Errorf("unknown metastore type '%s' (valid options: rdbms, dynamodb, memory)", opts.Metastore))
	}
}

func NewKMS(opts *Options, crypto appencryption.AEAD) appencryption.KeyManagementService {
	if opts.KMS == "static" {
		log.ErrorLog("*** WARNING WARNING WARNING USING STATIC MASTER KEY - THIS IS FOR TEST/DEBUG ONLY ***")

		m, err := kms.NewStatic("thisIsAStaticMasterKeyForTesting", aead.NewAES256GCM())
		if err != nil {
			log.ErrorLogf("PANIC: Failed to create static master key for KMS type 'static': %v", err.Error())
			panic(fmt.Errorf("failed to create static master key for KMS type 'static': %w", err))
		}

		return m
	} else if opts.KMS == "test-debug-static" {
		// We don't warn if the user specifically asks for test-debug-static
		m, err := kms.NewStatic("thisIsAStaticMasterKeyForTesting", crypto)
		if err != nil {
			log.ErrorLogf("PANIC: Failed to create static master key for KMS type 'test-debug-static': %v", err.Error())
			panic(fmt.Errorf("failed to create static master key for KMS type 'test-debug-static': %w", err))
		}

		return m
	}

	m, err := kms.NewAWS(crypto, opts.PreferredRegion, opts.RegionMap)
	if err != nil {
		log.ErrorLogf("PANIC: Failed to create AWS KMS with preferred region '%s': %v", opts.PreferredRegion, err.Error())
		panic(fmt.Errorf("failed to create AWS KMS with preferred region '%s': %w", opts.PreferredRegion, err))
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
