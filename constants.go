package main

const ERR_NOT_INITIALIZED = -100
const ERR_ALREADY_INITIALIZED = -101
const ERR_GET_SESSION_FAILED = -102
const ERR_ENCRYPT_FAILED = -103
const ERR_DECRYPT_FAILED = -104
const ERR_BAD_CONFIG = -105

const EstimatedEncryptionOverhead = 48
const EstimatedEnvelopeOverhead = 185
const Base64Overhead = 1.34
