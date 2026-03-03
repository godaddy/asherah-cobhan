package asherah

import "testing"

func TestRedactConnectionStringMysqlWithPassword(t *testing.T) {
	result := redactConnectionString("user:secret@tcp(localhost:3306)/db")
	expected := "user:***@tcp(localhost:3306)/db"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRedactConnectionStringMysqlWithoutPassword(t *testing.T) {
	result := redactConnectionString("user@tcp(localhost:3306)/db")
	expected := "user@tcp(localhost:3306)/db"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRedactConnectionStringPostgresWithPassword(t *testing.T) {
	result := redactConnectionString("postgres://user:secret@localhost:5432/db")
	expected := "postgres://user:***@localhost:5432/db"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRedactConnectionStringPostgresWithoutPassword(t *testing.T) {
	result := redactConnectionString("postgres://user@localhost:5432/db")
	expected := "postgres://user@localhost:5432/db"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRedactConnectionStringEmpty(t *testing.T) {
	result := redactConnectionString("")
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}
