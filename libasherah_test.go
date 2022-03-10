package main

import (
	"fmt"
	"testing"

	"github.com/godaddy/cobhan-go"
)

func validSetupForTesting(t *testing.T) {
	config := &Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.SessionCacheDuration = 1000
	config.SessionCacheMaxSize = 2
	config.ExpireAfter = 1000
	config.CheckInterval = 1000
	config.Verbose = true
	config.RegionMap = RegionMap{}
	config.RegionMap.UnmarshalFlag("region1=arn1,region2=arn2")

	buf := cobhan.AllocateBuffer(4096)
	result := cobhan.JsonToBufferSafe(config, &buf)
	if result != ERR_NONE {
		t.Errorf("BytesToBufferSafe returned %v", result)
	}

	SetupJson(cobhan.Ptr(&buf))
}

func testAllocateStringBuffer(t *testing.T, str string) []byte {
	buf, result := cobhan.AllocateStringBuffer(str)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("AllocateStringBuffer returned %v", result))
		t.FailNow()
	}
	return buf
}

func TestSetupJson(t *testing.T) {
	validSetupForTesting(t)
	Shutdown()
}

func TestSetupJson2(t *testing.T) {
	config := &Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.Verbose = false

	buf := cobhan.AllocateBuffer(4096)
	result := cobhan.JsonToBufferSafe(config, &buf)
	if result != ERR_NONE {
		t.Errorf("BytesToBufferSafe returned %v", result)
	}

	SetupJson(cobhan.Ptr(&buf))
	Shutdown()
}

func TestSetupJsonTwice(t *testing.T) {
	config := &Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.Verbose = true

	buf := cobhan.AllocateBuffer(4096)
	result := cobhan.JsonToBufferSafe(config, &buf)
	if result != ERR_NONE {
		t.Errorf("BytesToBufferSafe returned %v", result)
	}

	SetupJson(cobhan.Ptr(&buf))
	SetupJson(cobhan.Ptr(&buf))
	Shutdown()
}

func TestSetupInvalidJson(t *testing.T) {
	str := "}InvalidJson{"

	buf := cobhan.AllocateBuffer(1024)
	result := cobhan.StringToBufferSafe(str, &buf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("StringToBufferSafe returned %v", result))
	}

	SetupJson(cobhan.Ptr(&buf))
	Shutdown()
}

func TestSetupNullJson(t *testing.T) {
	SetupJson(nil)
	Shutdown()
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	result = Decrypt(cobhan.Ptr(&partitionId),
		cobhan.Ptr(&encryptedData),
		cobhan.Ptr(&encryptedKey),
		created,
		cobhan.Ptr(&parentKeyId),
		parentKeyCreated,
		cobhan.Ptr(&decryptedData),
	)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Decrypt returned %v", result))
	}

	output, result := cobhan.BufferToStringSafe(&decryptedData)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToStringSafe returned %v", result))
	}
	if output != input {
		t.Error(fmt.Sprintf("Expected %v Actual %v", input, output))
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
	validSetupForTesting(t)
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
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("Encrypt returned %v", result))
	}

	encryptedData[cobhan.BUFFER_HEADER_SIZE+4] = 1
	encryptedData[cobhan.BUFFER_HEADER_SIZE+5] = 2
	encryptedData[cobhan.BUFFER_HEADER_SIZE+6] = 3
	encryptedData[cobhan.BUFFER_HEADER_SIZE+7] = 4

	decryptedData := cobhan.AllocateBuffer(256)

	created, result := cobhan.BufferToInt64Safe(&createdBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
	}

	parentKeyCreated, result := cobhan.BufferToInt64Safe(&parentKeyCreatedBuf)
	if result != ERR_NONE {
		t.Error(fmt.Sprintf("BufferToInt64Safe returned %v", result))
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
