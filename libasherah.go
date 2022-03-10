package main

import (
	"C"
)
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

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

type AsherahConfig struct {
	KmsType                string `json:"kmsType"`
	Metastore              string `json:"metastore"`
	ServiceName            string `json:"serviceName"`
	ProductID              string `json:"productId"`
	ConnectionString       string `json:"rdbmsConnectionString,omitempty"`
	ReplicaReadConsistency string `json:"replicaReadConsistency,omitempty"`
	DynamoDBEndpoint       string `json:"dynamoDbEndpoint,omitempty"`
	DynamoDBRegion         string `json:"dynamoDbRegion,omitempty"`
	DynamoDBTableName      string `json:"dynamoDbTableName,omitempty"`
	EnableRegionSuffix     bool   `json:"enableRegionSuffix"`
	PreferredRegion        string `json:"preferredRegion,omitempty"`
	RegionMapStr           string `json:"regionMapStr,omitempty"`
	SessionCacheMaxSize    int    `json:"sessionCacheMaxSize,omitempty"`
	SessionCacheDuration   int    `json:"sessionCacheDuration,omitempty"`
	ExpireAfter            int    `json:"expireAfter,omitempty"`
	CheckInterval          int    `json:"checkInterval,omitempty"`
	Verbose                bool   `json:"verbose"`
	SessionCache           bool   `json:"sessionCache"`
	DebugOutput            bool   `json:"debugOutput"`
}

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

	var result int32
	config := AsherahConfig{}

	configJsonStr, result := cobhan.BufferToString(configJson)
	if result != ERR_NONE {
		StdoutDebugOutput("Failed to deserialize configuration string")
		return result
	}

	err := json.Unmarshal([]byte(configJsonStr), &config)
	if err != nil {
		StdoutDebugOutput("Failed to deserialize configuration JSON: " + err.Error())
		return cobhan.ERR_JSON_DECODE_FAILED
	}

	if config.DebugOutput {
		globalDebugOutput = StdoutDebugOutput
		globalDebugOutput("Enabled debug output")
	} else {
		globalDebugOutput = NullDebugOutput
	}

	globalDebugOutput("Successfully deserialized config JSON")
	globalDebugOutput(config)

	return setupAsherah(config)
}

func setupAsherah(config AsherahConfig) int32 {
	options := &Options{}
	options.KMS = config.KmsType             // "kms"
	options.ServiceName = config.ServiceName // "chatterbox"
	options.ProductID = config.ProductID     //"facebook"
	options.Verbose = config.Verbose
	options.EnableSessionCaching = config.SessionCache
	options.Metastore = config.Metastore //"dynamodb"
	crypto := aead.NewAES256GCM()
	options.ConnectionString = config.ConnectionString
	options.ReplicaReadConsistency = config.ReplicaReadConsistency
	options.DynamoDBEndpoint = config.DynamoDBEndpoint
	options.DynamoDBRegion = config.DynamoDBRegion
	options.DynamoDBTableName = config.DynamoDBTableName
	options.EnableRegionSuffix = config.EnableRegionSuffix
	options.PreferredRegion = config.PreferredRegion

	if config.SessionCacheMaxSize == 0 {
		options.SessionCacheMaxSize = appencryption.DefaultSessionCacheMaxSize
	} else {
		options.SessionCacheMaxSize = config.SessionCacheMaxSize
	}

	if config.SessionCacheDuration == 0 {
		options.SessionCacheDuration = appencryption.DefaultSessionCacheDuration
	} else {
		options.SessionCacheDuration = time.Second * time.Duration(config.SessionCacheDuration)
	}

	if config.ExpireAfter == 0 {
		options.ExpireAfter = appencryption.DefaultExpireAfter
	} else {
		options.ExpireAfter = time.Second * time.Duration(config.ExpireAfter)
	}

	if config.CheckInterval == 0 {
		options.CheckInterval = appencryption.DefaultRevokedCheckInterval
	} else {
		options.CheckInterval = time.Second * time.Duration(config.CheckInterval)
	}

	if len(config.RegionMapStr) > 0 {
		regionMap := make(map[string]string)
		pairs := strings.Split(config.RegionMapStr, ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) != 2 || len(parts[1]) == 0 {
				globalDebugOutput("Error: RegionMap must be in the form of REGION1=ARN1[,REGION2=ARN2]")
				return ERR_BAD_CONFIG
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

	globalDebugOutput("Decrypting with partition: " + partitionId)

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

	globalDebugOutput("parentKeyId: " + parentKeyId)

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
		globalDebugOutput("Decrypt failed: " + err.Error())
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

	globalDebugOutput("Encrypting with partition: " + partitionId)

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != ERR_NONE {
		return result
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutput("Encrypt: GetSession failed: " + err.Error())
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
		globalDebugOutput(fmt.Sprintf("Encrypted data length: %v", len(drr.Data)))
		globalDebugOutput(fmt.Sprintf("Encrypt: BytesToBuffer returned %v for outputEncryptedDataPtr", result))
		return result
	}

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("Encrypt: BytesToBuffer returned %v for outputEncryptedKeyPtr", result))
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("Encrypt: Int64ToBuffer returned %v for outputCreatedPtr", result))
		return result
	}

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("Encrypt: BytesToBuffer returned %v for outputParentKeyIdPtr", result))
		return result
	}

	globalDebugOutput("Encrypting with parent key ID: " + drr.Key.ParentKeyMeta.ID)

	result = cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("Encrypt: BytesToBuffer returned %v for outputParentKeyCreatedPtr", result))
		return result
	}

	return ERR_NONE
}

//export EncryptJson
func EncryptJson(partitionIdPtr unsafe.Pointer, inputPtr unsafe.Pointer, outputPtr unsafe.Pointer) int32 {
	if globalInitialized == 0 {
		return ERR_NOT_INITIALIZED
	}

	var result int32
	globalDebugOutput("EncryptJson()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutput("Encrypting with partition: " + partitionId)

	input, result := cobhan.BufferToBytes(inputPtr)
	if result != ERR_NONE {
		return result
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutput("EncryptJson: GetSession failed: " + err.Error())
		return ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	ctx := context.Background()
	drr, err := session.Encrypt(ctx, input)
	if err != nil {
		globalDebugOutput("EncryptJson failed: " + err.Error())
		return ERR_ENCRYPT_FAILED
	}

	result = cobhan.JsonToBuffer(drr, outputPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("EncryptJson: JsonToBuffer returned %v for outputPtr", result))
		return result
	}

	return ERR_NONE
}

//export DecryptJson
func DecryptJson(partitionIdPtr unsafe.Pointer, inputPtr unsafe.Pointer, outputPtr unsafe.Pointer) int32 {
	if globalInitialized == 0 {
		return ERR_NOT_INITIALIZED
	}

	globalDebugOutput("DecryptJson()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutput("Decrypting with partition: " + partitionId)

	jsonData, result := cobhan.BufferToString(inputPtr)
	if result != ERR_NONE {
		return result
	}

	var drr appencryption.DataRowRecord
	err := json.Unmarshal([]byte(jsonData), &drr)
	if err != nil {
		StdoutDebugOutput("Failed to deserialize input JSON: " + err.Error())
		return cobhan.ERR_JSON_DECODE_FAILED
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutput(err.Error())
		return ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	ctx := context.Background()
	output, err := session.Decrypt(ctx, drr)
	if err != nil {
		globalDebugOutput("DecryptJson failed: " + err.Error())
		return ERR_DECRYPT_FAILED
	}

	result = cobhan.BytesToBuffer(output, outputPtr)
	if result != ERR_NONE {
		globalDebugOutput(fmt.Sprintf("DecryptJson: BytesToBuffer returned %v for outputPtr", result))
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
