package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/godaddy/asherah-cobhan/internal/asherah"
	"github.com/godaddy/cobhan-go"
)

var Verbose = true

func setupAsherahForTesting(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.SessionCacheDuration = 1000
	config.SessionCacheMaxSize = 2
	config.ExpireAfter = 1000
	config.CheckInterval = 1000
	config.Verbose = Verbose
	config.RegionMap = asherah.RegionMap{}
	config.RegionMap["region1"] = "arn1"
	config.RegionMap["region2"] = "arn2"

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
}

func testAllocateStringBuffer(t *testing.T, str string) []byte {
	buf, result := cobhan.AllocateStringBuffer(str)
	if result != cobhan.ERR_NONE {
		t.Errorf("AllocateStringBuffer returned %v", result)
	}
	return buf
}

func testAllocateBytesBuffer(t *testing.T, bytes []byte) []byte {
	buf, result := cobhan.AllocateBytesBuffer(bytes)
	if result != cobhan.ERR_NONE {
		t.Errorf("AllocateStringBuffer returned %v", result)
	}
	return buf
}

func testAllocateJsonBuffer(t *testing.T, obj interface{}) []byte {
	bytes, err := json.Marshal(obj)
	if err != nil {
		t.Errorf("json.Marshal returned %v", err)
	}
	return testAllocateBytesBuffer(t, bytes)
}

func TestSetupJson(t *testing.T) {
	setupAsherahForTesting(t)
	Shutdown()
}

func TestSetupJsonAlternateConfiguration(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.Verbose = Verbose

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
	Shutdown()
}

func TestSetupJsonRdbmWithMysqlDefaultDbType(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "rdbms"
	config.ConnectionString = "user@tcp(localhost:3306)/db"
	config.EnableSessionCaching = true
	config.Verbose = Verbose

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
	Shutdown()
}

func TestSetupJsonRdbmWithMysqlDbType(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "rdbms"
	config.ConnectionString = "user@tcp(localhost:3306)/db"
	config.SQLMetastoreDBType = "mysql"
	config.EnableSessionCaching = true
	config.Verbose = Verbose

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
	Shutdown()
}

func TestSetupJsonRdbmWithPostgresDbType(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "rdbms"
	config.ConnectionString = "postgres://user@localhost:5432/db"
	config.SQLMetastoreDBType = "postgres"
	config.EnableSessionCaching = true
	config.Verbose = Verbose

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
	Shutdown()
}

func TestSetupJsonTwice(t *testing.T) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.Verbose = Verbose

	buf := testAllocateJsonBuffer(t, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetupJson returned %v", result)
	}
	defer Shutdown()
	result = SetupJson(cobhan.Ptr(&buf))
	if result != ERR_ALREADY_INITIALIZED {
		t.Errorf("Expected SetupJson to return ERR_ALREADY_INITIALIZED got %v", result)
	}
}

func TestSetupInvalidJson(t *testing.T) {
	str := "}InvalidJson{"

	buf := testAllocateStringBuffer(t, str)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_JSON_DECODE_FAILED {
		t.Errorf("Expected SetupJson to return ERR_JSON_DECODE_FAILED got %v", result)
	}
	Shutdown()
}

func TestSetEnv(t *testing.T) {
	setupAsherahForTesting(t)
	env := Env{}
	env["VAR1"] = "VALUE1"
	env["VAR2"] = "VALUE2"

	buf := testAllocateJsonBuffer(t, env)

	result := SetEnv(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		t.Errorf("SetEnv returned %v", result)
	}

	var1 := os.Getenv("VAR1")
	if var1 != "VALUE1" {
		t.Errorf("Failed to set VAR1 env var")
	}

	var2 := os.Getenv("VAR2")
	if var2 != "VALUE2" {
		t.Errorf("Failed to set VAR2 env var")
	}
	Shutdown()
}

func TestSetupInvalidEnv(t *testing.T) {
	str := "}InvalidEnv{"

	buf := testAllocateStringBuffer(t, str)

	result := SetEnv(cobhan.Ptr(&buf))
	if result != cobhan.ERR_JSON_DECODE_FAILED {
		t.Errorf("Expected SetEnv to return ERR_JSON_DECODE_FAILED got %v", result)
	}
	Shutdown()
}

func TestSetupNullJson(t *testing.T) {
	SetupJson(nil)
	Shutdown()
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Decrypt returned %v", result)
	}

	output, result := cobhan.BufferToStringSafe(&decryptedData)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToStringSafe returned %v", result)
	}
	if output != input {
		t.Errorf("Expected %v Actual %v", input, output)
	}
}

func TestEncryptWithoutInit(t *testing.T) {
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != ERR_NOT_INITIALIZED {
		t.Error("Encrypt didn't return ERR_NOT_INITIALIZED")
	}
}

func TestDecryptWithoutInit(t *testing.T) {

	partitionId := testAllocateStringBuffer(t, "Partition")
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	parentKeyId := cobhan.AllocateBuffer(256)

	decryptedData := cobhan.AllocateBuffer(256)

	result := Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		1234,
		cobhan.Ptr(&parentKeyId),
		1234,
		cobhan.Ptr(&decryptedData),
	)
	if result != ERR_NOT_INITIALIZED {
		t.Error("Decrypt didn't return ERR_NOT_INITIALIZED")
	}
}

func TestEncryptNullPartitionId(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	data := testAllocateStringBuffer(t, "InputData")

	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(nil,
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullData(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		nil,
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullEncryptedData(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		nil,
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullEncryptedKey(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedData := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		nil,
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullCreatedBuf(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedKey := cobhan.AllocateBuffer(256)
	encryptedData := cobhan.AllocateBuffer(256)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		nil,
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullParentKeyId(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedKey := cobhan.AllocateBuffer(256)
	encryptedData := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		nil,
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestEncryptNullParentKeyCreated(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, "InputData")
	encryptedKey := cobhan.AllocateBuffer(256)
	encryptedData := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		nil,
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Encrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptNullPartitionId(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(nil,
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Decrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptNullEncryptedData(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		nil,
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Decrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptNullEncryptedKey(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		nil,
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Decrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptNullParentKeyId(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		nil,
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Decrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptNullDecryptedData(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		nil,
	)
	if result != cobhan.ERR_NULL_PTR {
		t.Error("Decrypt didn't return ERR_NULL_PTR")
	}
}

func TestDecryptBadData(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	input := "InputData"
	partitionId := testAllocateStringBuffer(t, "Partition")
	data := testAllocateStringBuffer(t, input)
	encryptedData := cobhan.AllocateBuffer(256)
	encryptedKey := cobhan.AllocateBuffer(256)
	createdBuf := cobhan.AllocateBuffer(8)
	parentKeyId := cobhan.AllocateBuffer(256)
	parentKeyCreatedBuf := cobhan.AllocateBuffer(8)

	result := Encrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&data),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		cobhan.Ptr(&createdBuf),
		cobhan.Ptr(&parentKeyId),
		cobhan.Ptr(&parentKeyCreatedBuf),
	)
	if result != cobhan.ERR_NONE {
		t.Errorf("Encrypt returned %v", result)
	}

	// Intentionally corrupt the encrypted data
	encryptedData[cobhan.BUFFER_HEADER_SIZE+4] = 1
	encryptedData[cobhan.BUFFER_HEADER_SIZE+5] = 2
	encryptedData[cobhan.BUFFER_HEADER_SIZE+6] = 3
	encryptedData[cobhan.BUFFER_HEADER_SIZE+7] = 4

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToInt64Safe returned %v", result)
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != ERR_DECRYPT_FAILED {
		t.Error("Decrypt didn't return ERR_DECRYPT_FAILED")
	}
}

func TestEncryptToJsonAndDecryptFromJsonCycle(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	cycleEncryptToJsonAndDecryptFromJson("1", "1", t)
	cycleEncryptToJsonAndDecryptFromJson("InputString", "Partition", t)
}

func TestEncryptToJsonAndDecryptFromJsonCycleLong(t *testing.T) {
	setupAsherahForTesting(t)
	defer Shutdown()

	longString := strings.Repeat("X", 16384)
	cycleEncryptToJsonAndDecryptFromJson(longString, "Partition", t)
	cycleEncryptToJsonAndDecryptFromJson(longString, longString, t)

	longerString := strings.Repeat("X", 2097152)
	cycleEncryptToJsonAndDecryptFromJson(longerString, "Partition", t)
	cycleEncryptToJsonAndDecryptFromJson(longerString, longerString, t)
}

func cycleEncryptToJsonAndDecryptFromJson(input string, partition string, t *testing.T) {
	partitionIdBuf := testAllocateStringBuffer(t, partition)
	inputBuf := testAllocateStringBuffer(t, input)

	estimatedBufferSize := EstimateBufferInt(len(input), len(partition))
	t.Logf("Estimated buffer size: %v", estimatedBufferSize)

	encryptedDataBuf := cobhan.AllocateBuffer(estimatedBufferSize)

	result := EncryptToJson(cobhan.Ptr(&partitionIdBuf), cobhan.Ptr(&inputBuf), cobhan.Ptr(&encryptedDataBuf))
	if result != cobhan.ERR_NONE {
		t.Errorf("EncryptToJson returned %v", result)
		return
	}

	encrypted_data, result := cobhan.BufferToString(cobhan.Ptr(&encryptedDataBuf))
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToString returned %v", result)
		return
	}

	encryptedDataInputBuf := testAllocateStringBuffer(t, encrypted_data)

	decryptedDataBuf := cobhan.AllocateBuffer(len(encrypted_data))
	result = DecryptFromJson(cobhan.Ptr(&partitionIdBuf), cobhan.Ptr(&encryptedDataInputBuf), cobhan.Ptr(&decryptedDataBuf))
	if result != cobhan.ERR_NONE {
		t.Errorf("DecryptFromJson returned %v", result)
		return
	}

	decryptedData, result := cobhan.BufferToString(cobhan.Ptr(&decryptedDataBuf))
	if result != cobhan.ERR_NONE {
		t.Errorf("BufferToString returned %v", result)
		return
	}

	if decryptedData != input {
		t.Errorf("decryptedData %v does not match inputData data %v", decryptedData, input)
		return
	}
}
