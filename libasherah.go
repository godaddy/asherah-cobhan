package main

import (
	"C"
)
import (
	"context"
	"strings"

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

func main() {
}

var globalSessionFactory *appencryption.SessionFactory

//var globalCtx context.Context
var globalSession *appencryption.Session
var globalInitialized = false

func init() {
}

//export Setup
func Setup(kmsTypePtr unsafe.Pointer, metastorePtr unsafe.Pointer, rdbmsConnectionStringPtr unsafe.Pointer, dynamoDbEndpointPtr unsafe.Pointer, dynamoDbRegionPtr unsafe.Pointer,
	dynamoDbTableNamePtr unsafe.Pointer, enableRegionSuffixInt int32, serviceNamePtr unsafe.Pointer, productIdPtr unsafe.Pointer, preferredRegionPtr unsafe.Pointer, regionMapPtr unsafe.Pointer, verboseInt int32,
	sessionCacheInt int32) int32 {

	if globalInitialized {
		return ERR_ALREADY_INITIALIZED
	}

	kmsType, result := cobhan.BufferToString(kmsTypePtr)
	if result != 0 {
		return result
	}

	metastore, result := cobhan.BufferToString(metastorePtr)
	if result != 0 {
		return result
	}

	rdbmsConnectionString, result := cobhan.BufferToString(rdbmsConnectionStringPtr)
	if result != 0 {
		return result
	}

	dynamoDbEndpoint, result := cobhan.BufferToString(dynamoDbEndpointPtr)
	if result != 0 {
		return result
	}

	dynamoDbRegion, result := cobhan.BufferToString(dynamoDbRegionPtr)
	if result != 0 {
		return result
	}

	dynamoDbTableName, result := cobhan.BufferToString(dynamoDbTableNamePtr)
	if result != 0 {
		return result
	}

	enableRegionSuffix := enableRegionSuffixInt != 0

	serviceName, result := cobhan.BufferToString(serviceNamePtr)
	if result != 0 {
		return result
	}

	productId, result := cobhan.BufferToString(productIdPtr)
	if result != 0 {
		return result
	}

	preferredRegion, result := cobhan.BufferToString(preferredRegionPtr)
	if result != 0 {
		return result
	}

	regionMapStr, result := cobhan.BufferToString(regionMapPtr)
	if result != 0 {
		return result
	}

	verbose := verboseInt != 0

	sessionCache := sessionCacheInt != 0

	setupAsherah(kmsType, metastore, rdbmsConnectionString, dynamoDbEndpoint, dynamoDbRegion, dynamoDbTableName,
		enableRegionSuffix, serviceName, productId, preferredRegion, regionMapStr, verbose, sessionCache)

	return ERR_NONE
}

func setupAsherah(kmsType string, metaStore string, rdbmsConnectionString string, dynamoDbEndpoint string,
	dynamoDbRegion string, dynamoDbTableName string, enableRegionSuffix bool, serviceName string, productId string,
	preferredRegion string, regionMapStr string, verbose bool, sessionCache bool) {
	options := &Options{}
	options.KMS = kmsType             // "kms"
	options.ServiceName = serviceName // "chatterbox"
	options.ProductID = productId     //"facebook"
	options.Verbose = verbose
	options.EnableSessionCaching = sessionCache
	options.Metastore = metaStore //"dynamodb"
	crypto := aead.NewAES256GCM()
	options.ConnectionString = rdbmsConnectionString
	options.DynamoDBEndpoint = dynamoDbEndpoint
	options.DynamoDBRegion = dynamoDbRegion
	options.DynamoDBTableName = dynamoDbTableName
	options.EnableRegionSuffix = enableRegionSuffix
	options.PreferredRegion = preferredRegion

	if len(regionMapStr) > 0 {
		regionMap := make(map[string]string)
		pairs := strings.Split(regionMapStr, ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) != 2 || len(parts[1]) == 0 {
				panic("argument must be in the form of REGION1=ARN1[,REGION2=ARN2]")
			}
			region, arn := parts[0], parts[1]
			regionMap[region] = arn
		}

		options.RegionMap = regionMap
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
	globalInitialized = true
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
	if !globalInitialized {
		return ERR_NOT_INITIALIZED
	}

	DebugOutput("Decrypt()")
	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != 0 {
		return result
	}

	DebugOutput("Decrypting with partition: " + partitionId)

	encryptedData, result := cobhan.BufferToBytes(encryptedDataPtr)
	if result != 0 {
		return result
	}

	//DebugOutput("encryptedData length: " + string(len(encryptedData)))

	encryptedKey, result := cobhan.BufferToBytes(encryptedKeyPtr)
	if result != 0 {
		return result
	}

	//DebugOutput("encryptedKey length: " + string(len(encryptedKey)))

	parentKeyId, result := cobhan.BufferToString(parentKeyIdPtr)
	if result != 0 {
		return result
	}

	DebugOutput("parentKeyId: " + parentKeyId)

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		DebugOutput(err.Error())
		return ERR_GET_SESSION_FAILED
	}

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
		DebugOutput("Decrypt failed: " + err.Error())
		return ERR_DECRYPT_FAILED
	}

	return cobhan.BytesToBuffer(data, outputDecryptedDataPtr)
}

//export Encrypt
func Encrypt(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, outputEncryptedDataPtr unsafe.Pointer,
	outputEncryptedKeyPtr unsafe.Pointer, outputCreatedPtr unsafe.Pointer, outputParentKeyIdPtr unsafe.Pointer,
	outputParentKeyCreatedPtr unsafe.Pointer) int32 {
	if !globalInitialized {
		return ERR_NOT_INITIALIZED
	}

	DebugOutput("Encrypt()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != 0 {
		return result
	}

	DebugOutput("Encrypting with partition: " + partitionId)

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != 0 {
		return result
	}

	//DebugOutput("Encrypting with data length: " + string(len(data)))

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		DebugOutput(err.Error())
		return ERR_GET_SESSION_FAILED
	}

	ctx := context.Background()
	drr, err := session.Encrypt(ctx, data)
	if err != nil {
		DebugOutput("Encrypt failed: " + err.Error())
		return ERR_ENCRYPT_FAILED
	}

	result = cobhan.BytesToBuffer(drr.Data, outputEncryptedDataPtr)
	if result != 0 {
		return result
	}

	//DebugOutput("Encrypting with output encrypted data length: " + string(len(drr.Data)))

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != 0 {
		return result
	}

	//DebugOutput("Encrypting with output encrypted key length: " + string(len(drr.Key.EncryptedKey)))

	cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != 0 {
		return result
	}

	DebugOutput("Encrypting with parent key ID: " + drr.Key.ParentKeyMeta.ID)

	cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)

	return 0
}

func NewCryptoPolicy(options *Options) *appencryption.CryptoPolicy {
	//TODO: Add these variables to setup
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
