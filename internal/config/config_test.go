package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoad проверяет загрузку конфигурации из всех источников:
// флагов, переменных окружения и JSON-файла, а также значения по умолчанию.
// Вызывается один раз, т.к. повторная регистрация флагов приведёт к панике.
func TestLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	// log_level и bot_token в файле должны быть проигнорированы:
	// log_level задан флагом, bot_token — переменной окружения.
	content := `{"log_level": "WARN", "bot_token": "file-token", "count_speach": 4}`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("BOT_TOKEN", "env-token")

	// Сохраняем и подменяем аргументы командной строки для flag.Parse.
	oldArgs := os.Args
	os.Args = []string{
		oldArgs[0],
		"-l=DEBUG",
		"-n=9",
		"-q=3s",
		"-w=postgres://dsn",
		"-c=" + cfgPath,
	}
	defer func() { os.Args = oldArgs }()

	Load()

	c := New()

	// Приоритет: флаг > env > config-файл > default.
	if got := c.GetLogLevel(); got != "DEBUG" {
		t.Errorf("GetLogLevel() = %q, want %q (flag)", got, "DEBUG")
	}
	if got := c.GetCountChat(); got != 9 {
		t.Errorf("GetCountChat() = %d, want %d (flag)", got, 9)
	}
	if got := c.GetStatusRequestPeriod(); got != 3*time.Second {
		t.Errorf("GetStatusRequestPeriod() = %v, want %v (flag)", got, 3*time.Second)
	}
	if got := c.GetConnectionString(); got != "postgres://dsn" {
		t.Errorf("GetConnectionString() = %q, want %q (flag)", got, "postgres://dsn")
	}
	if got := c.GetBotToken(); got != "env-token" {
		t.Errorf("GetBotToken() = %q, want %q (env over config file)", got, "env-token")
	}
	if got := c.GetCountSpeach(); got != 4 {
		t.Errorf("GetCountSpeach() = %d, want %d (config file)", got, 4)
	}

	// Значения по умолчанию.
	if got := c.GetSpeachAuthHost(); got != defaultSpeechAuthAddress {
		t.Errorf("GetSpeachAuthHost() = %q, want %q (default)", got, defaultSpeechAuthAddress)
	}
	if got := c.GetSpeachRequestHost(); got != defaultSpeechRequestAddress {
		t.Errorf("GetSpeachRequestHost() = %q, want %q (default)", got, defaultSpeechRequestAddress)
	}
	if got := c.GetSpeachRQUID(); got != "" {
		t.Errorf("GetSpeachRQUID() = %q, want empty (default)", got)
	}
	if got := c.GetSpeachAuthKey(); got != "" {
		t.Errorf("GetSpeachAuthKey() = %q, want empty (default)", got)
	}
	if got := c.GetChatAuthHost(); got != defaultChatAuthAddress {
		t.Errorf("GetChatAuthHost() = %q, want %q (default)", got, defaultChatAuthAddress)
	}
	if got := c.GetChatRequestHost(); got != defaultChatRequestAddress {
		t.Errorf("GetChatRequestHost() = %q, want %q (default)", got, defaultChatRequestAddress)
	}
	if got := c.GetChatRQUID(); got != "" {
		t.Errorf("GetChatRQUID() = %q, want empty (default)", got)
	}
	if got := c.GetChatAuthKey(); got != "" {
		t.Errorf("GetChatAuthKey() = %q, want empty (default)", got)
	}
}

// TestGetString проверяет getString: корректное значение, отсутствие ключа и несовпадение типа.
func TestGetString(t *testing.T) {
	c := &Config{list: map[configType]*ConfigValue{
		configLogLevel: {valueType: valueString, value: "X"},
	}}
	if got := c.getString(configLogLevel); got != "X" {
		t.Errorf("getString() = %q, want %q", got, "X")
	}

	empty := &Config{list: map[configType]*ConfigValue{}}
	if got := empty.getString(configLogLevel); got != "" {
		t.Errorf("getString(missing) = %q, want empty", got)
	}

	wrongType := &Config{list: map[configType]*ConfigValue{
		configLogLevel: {valueType: valueInt, value: 5},
	}}
	if got := wrongType.getString(configLogLevel); got != "" {
		t.Errorf("getString(wrong type) = %q, want empty", got)
	}

	nilValue := &Config{list: map[configType]*ConfigValue{
		configLogLevel: {valueType: valueString, value: nil},
	}}
	if got := nilValue.getString(configLogLevel); got != "" {
		t.Errorf("getString(nil value) = %q, want empty", got)
	}
}

// TestGetBool проверяет getBool: корректное значение, отсутствие ключа и несовпадение типа.
func TestGetBool(t *testing.T) {
	c := &Config{list: map[configType]*ConfigValue{
		configLogLevel: {valueType: valueBool, value: true},
	}}
	if got := c.getBool(configLogLevel); got != true {
		t.Errorf("getBool() = %v, want true", got)
	}

	empty := &Config{list: map[configType]*ConfigValue{}}
	if got := empty.getBool(configLogLevel); got != false {
		t.Errorf("getBool(missing) = %v, want false", got)
	}

	wrongType := &Config{list: map[configType]*ConfigValue{
		configLogLevel: {valueType: valueString, value: "true"},
	}}
	if got := wrongType.getBool(configLogLevel); got != false {
		t.Errorf("getBool(wrong type) = %v, want false", got)
	}
}

// TestGetInt проверяет getInt: корректное значение, отсутствие ключа и несовпадение типа.
func TestGetInt(t *testing.T) {
	c := &Config{list: map[configType]*ConfigValue{
		configCountChat: {valueType: valueInt, value: 42},
	}}
	if got := c.getInt(configCountChat); got != 42 {
		t.Errorf("getInt() = %d, want 42", got)
	}

	empty := &Config{list: map[configType]*ConfigValue{}}
	if got := empty.getInt(configCountChat); got != 0 {
		t.Errorf("getInt(missing) = %d, want 0", got)
	}

	wrongType := &Config{list: map[configType]*ConfigValue{
		configCountChat: {valueType: valueString, value: "42"},
	}}
	if got := wrongType.getInt(configCountChat); got != 0 {
		t.Errorf("getInt(wrong type) = %d, want 0", got)
	}
}

// TestGetDuration проверяет getDuration: корректное значение, отсутствие ключа и несовпадение типа.
func TestGetDuration(t *testing.T) {
	c := &Config{list: map[configType]*ConfigValue{
		configStatusRequestPeriod: {valueType: valueDuration, value: 5 * time.Second},
	}}
	if got := c.getDuration(configStatusRequestPeriod); got != 5*time.Second {
		t.Errorf("getDuration() = %v, want %v", got, 5*time.Second)
	}

	empty := &Config{list: map[configType]*ConfigValue{}}
	if got := empty.getDuration(configStatusRequestPeriod); got != 0 {
		t.Errorf("getDuration(missing) = %v, want 0", got)
	}

	wrongType := &Config{list: map[configType]*ConfigValue{
		configStatusRequestPeriod: {valueType: valueInt, value: 5},
	}}
	if got := wrongType.getDuration(configStatusRequestPeriod); got != 0 {
		t.Errorf("getDuration(wrong type) = %v, want 0", got)
	}
}
