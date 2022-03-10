package main

import (
	"C"
)
import (
	"context"
	"sync/atomic"

	"github.com/godaddy/asherah/go/securememory/memguard"
	"github.com/godaddy/cobhan-go"

	"unsafe"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/godaddy/asherah/go/appencryption"
	"github.com/godaddy/asherah/go/appencryption/pkg/crypto/aead"
	"github.com/godaddy/asherah/go/appencryption/pkg/kms"
	"github.com/godaddy/asherah/go/appencryption/pkg/persistence"
)

const ERR_NONE = 0
const ERR_NOT_INITIALIZED = -100
const ERR_ALREADY_INITIALIZED = -101
const ERR_GET_SESSION_FAILED = -102
const ERR_ENCRYPT_FAILED = -103
const ERR_DECRYPT_FAILED = -104
const ERR_BAD_CONFIG = -105

func main() {
}

var globalSessionFactory *appencryption.SessionFactory
var globalInitialized int32 = 0
var globalDebugOutput func(interface{}) = nil
var globalDebugOutputf func(format string, args ...interface{}) = nil

//export Shutdown
func Shutdown() {
	if globalInitialized != 0 {
		globalDebugOutput("Shutting down Asherah")
		globalSessionFactory.Close()
		globalSessionFactory = nil
		atomic.StoreInt32(&globalInitialized, 0)
	}
}

//export SetupJson
func SetupJson(configJson unsafe.Pointer) int32 {
	if globalInitialized != 0 {
		return ERR_ALREADY_INITIALIZED
	}

	options := &Options{}
	result := cobhan.BufferToJsonStruct(configJson, options)
	if result != ERR_NONE {
		StdoutDebugOutput("Failed to deserialize configuration string")
		return result
	}

	if options.Verbose {
		globalDebugOutput = StdoutDebugOutput
		globalDebugOutputf = StdoutDebugOutputf
		globalDebugOutput("Enabled debug output")
	} else {
		globalDebugOutput = NullDebugOutput
		globalDebugOutputf = StdoutDebugOutputf
	}

	globalDebugOutput("Successfully deserialized config JSON")
	globalDebugOutput(options)

	return setupAsherah(options)
}

func setupAsherah(options *Options) int32 {
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

	atomic.StoreInt32(&globalInitialized, 1)
	globalDebugOutput("Successfully configured Asherah")
	return ERR_NONE
}

func NewMetastore(opts *Options) appencryption.Metastore {
	switch opts.Metastore {
	case "rdbms":
		// TODO: support other databases
		db, err := newMysql(opts.ConnectionString)
		if err != nil {
			panic(err)
		}

		// set optional replica read consistency
		if len(opts.ReplicaReadConsistency) > 0 {
			err := setRdbmsReplicaReadConsistencyValue(opts.ReplicaReadConsistency)
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

func NewKMS(opts *Options, crypto appencryption.AEAD) appencryption.KeyManagementService {
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

//export Decrypt
func Decrypt(partitionIdPtr unsafe.Pointer, encryptedDataPtr unsafe.Pointer, encryptedKeyPtr unsafe.Pointer,
	created int64, parentKeyIdPtr unsafe.Pointer, parentKeyCreated int64, outputDecryptedDataPtr unsafe.Pointer) int32 {
	if globalInitialized == 0 {
		return ERR_NOT_INITIALIZED
	}

	globalDebugOutput("Decrypt()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutputf("Decrypting with partition: %v", partitionId)

	encryptedData, result := cobhan.BufferToBytes(encryptedDataPtr)
	if result != ERR_NONE {
		return result
	}

	encryptedKey, result := cobhan.BufferToBytes(encryptedKeyPtr)
	if result != ERR_NONE {
		return result
	}

	parentKeyId, result := cobhan.BufferToString(parentKeyIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutputf("parentKeyId: %v", parentKeyId)

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutput(err.Error())
		return ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	drr := &appencryption.DataRowRecord{
		Data: encryptedData,
		Key: &appencryption.EnvelopeKeyRecord{
			EncryptedKey: encryptedKey,
			Created:      created,
			ParentKeyMeta: &appencryption.KeyMeta{
				ID:      parentKeyId,
				Created: parentKeyCreated,
			},
		},
	}

	ctx := context.Background()
	data, err := session.Decrypt(ctx, *drr)
	if err != nil {
		globalDebugOutputf("Decrypt failed: %v", err)
		return ERR_DECRYPT_FAILED
	}

	return cobhan.BytesToBuffer(data, outputDecryptedDataPtr)
}

//export Encrypt
func Encrypt(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, outputEncryptedDataPtr unsafe.Pointer,
	outputEncryptedKeyPtr unsafe.Pointer, outputCreatedPtr unsafe.Pointer, outputParentKeyIdPtr unsafe.Pointer,
	outputParentKeyCreatedPtr unsafe.Pointer) int32 {
	if globalInitialized == 0 {
		return ERR_NOT_INITIALIZED
	}

	globalDebugOutput("Encrypt()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutputf("Encrypting with partition: %v", partitionId)

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != ERR_NONE {
		return result
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutputf("Encrypt: GetSession failed: %v", err)
		return ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	ctx := context.Background()
	drr, err := session.Encrypt(ctx, data)
	if err != nil {
		globalDebugOutput("Encrypt failed: " + err.Error())
		return ERR_ENCRYPT_FAILED
	}

	result = cobhan.BytesToBuffer(drr.Data, outputEncryptedDataPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypted data length: %v", len(drr.Data))
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedDataPtr", result)
		return result
	}

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedKeyPtr", result)
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: Int64ToBuffer returned %v for outputCreatedPtr", result)
		return result
	}

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyIdPtr", result)
		return result
	}

	globalDebugOutput("Encrypting with parent key ID: " + drr.Key.ParentKeyMeta.ID)

	result = cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyCreatedPtr", result)
		return result
	}

	return ERR_NONE
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
