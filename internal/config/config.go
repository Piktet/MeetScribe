/*
Package config — управление конфигурацией приложения.

Конфигурация загружается из трёх источников в порядке приоритета:
1. Файл конфигурации (JSON)
2. Переменные окружения
3. Флаги командной строки
4. Значения по умолчанию

Флаг -a — адрес авторизации SaluteSpeech.
Флаг -b — адрес запросов SaluteSpeech.
Флаг -l — уровень логирования (по умолчанию "Info").
Флаг -c/-config — путь до JSON-файла конфигурации (по умолчанию "").
*/
package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"sync"
	"time"
)

// sourceType определяет источник значения конфигурации.
type sourceType int

const (
	_             sourceType = iota
	sourceEnv                // значение из переменной окружения
	sourceFlag               // значение из флага командной строки
	sourceConfig             // значение из JSON-файла
	sourceDefault            // значение по умолчанию
)

// valueTypeType определяет тип значения конфигурации.
type valueTypeType int

const (
	_             valueTypeType = iota
	valueString                 // строковое значение
	valueBool                   // булево значение
	valueInt                    // целочисленное значение
	valueDuration               // значение длительности
)

// ConfigValue — параметр конфигурации.
// Хранит значение, тип, источник и описание параметра.
type ConfigValue struct {
	defaultValue any           // значение по умолчанию
	flagName     []string      // имена флагов командной строки
	envName      string        // имя переменной окружения
	description  string        // описание параметра
	value        any           // текущее значение
	valueType    valueTypeType // тип значения
	source       sourceType    // источник значения
}

type configType string

const (
	configSpeechAuthAddress    = "speach_auth_address"
	configSpeechRequestAddress = "speach_request_address"
	configSpeechClientID       = "speach_client_id"
	configSpeechScope          = "speach_scope"
	configSpeechAuthKey        = "speach_auth_key"

	configChatAuthAddress    = "chat_auth_address"
	configChatRequestAddress = "chat_request_address"
	configChatClientID       = "chat_client_id"
	configChatScope          = "chat_scope"
	configChatAuthKey        = "chat_auth_key"

	configDBConnAddress = "database_dsn"

	configBotToken            = "bot_token"
	configCountChat           = "count_chat"
	configCountSpeach         = "count_speach"
	configStatusRequestPeriod = "status_request_period"

	configLogLevel   = "log_level"
	configConfigFile = "config_file"
)

const (
	defaultSpeechAuthAddress    = "ngw.devices.sberbank.ru:9443"
	defaultSpeechRequestAddress = "smartspeech.sber.ru"
	defaultSpeechClientID       = ""
	defaultSpeechScope          = ""
	defaultSpeechAuthKey        = ""
	defaultChatAuthAddress      = "ngw.devices.sberbank.ru:9443"
	defaultChatRequestAddress   = "gigachat.devices.sberbank.ru"
	defaultChatClientID         = ""
	defaultChatScope            = ""
	defaultChatAuthKey          = ""

	defaultBotToken            = ""
	defaultCountChat           = 1
	defaultCountSpeach         = 3
	defaultStatusRequestPeriod = time.Second

	defaultLogLevel = "INFO"
)

var (
	configData = map[configType]*ConfigValue{
		configSpeechAuthAddress:    {defaultValue: defaultSpeechAuthAddress, flagName: []string{"a"}, envName: "SPEECH_AUTH_ADDRESS", description: "адрес авторизации SaluteSpeech", valueType: valueString},
		configSpeechRequestAddress: {defaultValue: defaultSpeechRequestAddress, flagName: []string{"b"}, envName: "SPEECH_REQUEST_ADDRESS", description: "адрес запросов SaluteSpeech", source: sourceDefault, valueType: valueString},
		configSpeechClientID:       {defaultValue: defaultSpeechClientID, flagName: []string{"r"}, envName: "SPEECH_CLIENT_ID", description: "client_id SaluteSpeech", source: sourceDefault, valueType: valueString},
		configSpeechScope:          {defaultValue: defaultSpeechScope, flagName: []string{"d"}, envName: "SPEECH_SCOPE", description: "scope SaluteSpeech", source: sourceDefault, valueType: valueString},
		configSpeechAuthKey:        {defaultValue: defaultSpeechAuthKey, flagName: []string{"e"}, envName: "SPEECH_AUTH_KEY", description: "auth_key SaluteSpeech", source: sourceDefault, valueType: valueString},
		configChatAuthAddress:      {defaultValue: defaultChatAuthAddress, flagName: []string{"f"}, envName: "CHAT_AUTH_ADDRESS", description: "адрес авторизации GigaChat", valueType: valueString},
		configChatRequestAddress:   {defaultValue: defaultChatRequestAddress, flagName: []string{"g"}, envName: "CHAT_REQUEST_ADDRESS", description: "адрес запросов GigaChat", source: sourceDefault, valueType: valueString},
		configChatClientID:         {defaultValue: defaultChatClientID, flagName: []string{"h"}, envName: "CHAT_CLIENT_ID", description: "client_id GigaChat", source: sourceDefault, valueType: valueString},
		configChatScope:            {defaultValue: defaultChatScope, flagName: []string{"j"}, envName: "CHAT_SCOPE", description: "scope GigaChat", source: sourceDefault, valueType: valueString},
		configChatAuthKey:          {defaultValue: defaultChatAuthKey, flagName: []string{"k"}, envName: "CHAT_AUTH_KEY", description: "auth_key GigaChat", source: sourceDefault, valueType: valueString},

		configBotToken:            {defaultValue: defaultBotToken, flagName: []string{"m"}, envName: "BOT_TOKEN", description: "токен Telegram-бота", source: sourceDefault, valueType: valueString},
		configCountChat:           {defaultValue: defaultCountChat, flagName: []string{"n"}, envName: "COUNT_CHAT", description: "количество потоков чата", source: sourceDefault, valueType: valueInt},
		configCountSpeach:         {defaultValue: defaultCountSpeach, flagName: []string{"p"}, envName: "COUNT_SPEECH", description: "количество потоков речи", source: sourceDefault, valueType: valueInt},
		configStatusRequestPeriod: {defaultValue: defaultStatusRequestPeriod, flagName: []string{"q"}, envName: "STATUS_REQUEST_PERIOD", description: "период опроса статуса", source: sourceDefault, valueType: valueDuration},

		configLogLevel:   {defaultValue: defaultLogLevel, flagName: []string{"l"}, envName: "LOG_LEVEL", description: "уровень логирования", valueType: valueString},
		configConfigFile: {defaultValue: "", flagName: []string{"c", "config"}, envName: "CONFIG", description: "файл конфигурации", valueType: valueString},
	}
)

// Config — структура для хранения конфигурации приложения.
// Инициализируется один раз при первом вызове New() благодаря sync.Once.
type Config struct {
	list map[configType]*ConfigValue
}

var fn sync.Once

// New создает и инициализирует структуру с конфигурацией.
// Конфигурация загружается из флагов, переменных окружения, JSON-файла
// и значений по умолчанию с приоритетом: файл > env > флаг > дефолт.
// Функция потокобезопасна — инициализация происходит только один раз.
func New() *Config {

	fn.Do(func() {

		for _, v := range configData {
			v.value = v.defaultValue
			v.source = sourceDefault
			switch v.valueType {

			case valueString:
				for _, f := range v.flagName {
					_ = flag.String(f, v.defaultValue.(string), v.description)
				}
			case valueBool:
				for _, f := range v.flagName {
					_ = flag.Bool(f, v.defaultValue.(bool), v.description)
				}
			case valueInt:
				for _, f := range v.flagName {
					_ = flag.Int(f, v.defaultValue.(int), v.description)
				}
			case valueDuration:
				for _, f := range v.flagName {
					_ = flag.Duration(f, v.defaultValue.(time.Duration), v.description)
				}
			}
			if respEnv, ok := os.LookupEnv(v.envName); ok && respEnv != "" {
				v.value = respEnv
				v.source = sourceEnv
			}
		}

		flag.Parse()

		flag.Visit(func(flagValue *flag.Flag) {
			if v, ok := configData[configType(flagValue.Name)]; ok {
				v.source = sourceFlag
				switch v.valueType {
				case valueString:
					v.value = flagValue.Value.String()
				case valueBool:
					v.value = flagValue.Value.String() == "true"
				case valueInt:
					v.value, _ = strconv.Atoi(flagValue.Value.String())
				case valueDuration:
					val, _ := strconv.ParseInt(flagValue.Value.String(), 10, 64)
					v.value = time.Duration(val)
				}
			}
		})

		if confFile, ok := configData[configConfigFile]; ok {
			if data, err := os.ReadFile(confFile.value.(string)); err == nil {
				val := make(map[string]any)
				if err = json.Unmarshal(data, &val); err == nil {
					for k, v := range val {
						if item, ok := configData[configType(k)]; ok {
							if item.source == sourceDefault {
								item.source = sourceConfig
								switch item.valueType {
								case valueString:
									item.value = v.(string)
								case valueBool:
									item.value = v.(bool)
								case valueInt:
									item.value = v.(int)
								case valueDuration:
									item.value = v.(time.Duration)
								}
							}
						}
					}
				}
			}
		}
	})
	return &Config{list: configData}
}

func (c *Config) getString(name configType) string {
	if v, ok := c.list[name]; ok && v != nil && v.valueType == valueString && v.value != nil {
		return v.value.(string)
	}
	return ""
}

func (c *Config) getBool(name configType) bool {
	if v, ok := c.list[name]; ok && v != nil && v.valueType == valueBool && v.value != nil {
		return v.value.(bool)
	}
	return false
}

func (c *Config) getInt(name configType) int {
	if v, ok := c.list[name]; ok && v != nil && v.valueType == valueInt && v.value != nil {
		return v.value.(int)
	}
	return 0
}

func (c *Config) getDuration(name configType) time.Duration {
	if v, ok := c.list[name]; ok && v != nil && v.valueType == valueDuration && v.value != nil {
		return v.value.(time.Duration)
	}
	return 0
}

// GetConnectionString — возвращает строку подключения к PostgreSQL.
func (c *Config) GetConnectionString() string {
	return c.getString(configDBConnAddress)
}

// GetSpeachAuthHost — возвращает хост для авторизации в SaluteSpeech API.
func (c *Config) GetSpeachAuthHost() string {
	return c.getString(configSpeechAuthAddress)
}

// GetSpeachRequestHost — возвращает хост для запросов к SaluteSpeech API.
func (c *Config) GetSpeachRequestHost() string {
	return c.getString(configSpeechRequestAddress)
}

// GetSpeachRQUID — возвращает RQ UID для авторизации SaluteSpeech API.
func (c *Config) GetSpeachRQUID() string {
	return c.getString(configSpeechClientID)
}

// GetSpeachAuthKey — возвращает ключ авторизации для SaluteSpeech API.
func (c *Config) GetSpeachAuthKey() string {
	return c.getString(configSpeechAuthKey)
}

// GetChatAuthHost — возвращает хост для авторизации в GigaChat API.
func (c *Config) GetChatAuthHost() string {
	return c.getString(configChatAuthAddress)
}

// GetChatRequestHost — возвращает хост для запросов к GigaChat API.
func (c *Config) GetChatRequestHost() string {
	return c.getString(configChatRequestAddress)
}

// GetChatRQUID — возвращает RQ UID для авторизации GigaChat API.
func (c *Config) GetChatRQUID() string {
	return c.getString(configChatClientID)
}

// GetChatAuthKey — возвращает ключ авторизации для GigaChat API.
func (c *Config) GetChatAuthKey() string {
	return c.getString(configChatAuthKey)
}

// GetBotToken — возвращает токен Telegram-бота.
func (c *Config) GetBotToken() string {
	return c.getString(configBotToken)
}

// GetLogLevel — возвращает уровень логирования.
func (c *Config) GetLogLevel() string {
	return c.getString(configLogLevel)
}

// GetStatusRequestPeriod — возвращает период опроса статуса задачи.
func (c *Config) GetStatusRequestPeriod() time.Duration {
	return c.getDuration(configStatusRequestPeriod)
}

// GetCountSpeach — возвращает количество потоков для обработки речи.
func (c *Config) GetCountSpeach() int {
	return c.getInt(configCountSpeach)
}

// GetCountChat — возвращает количество потоков для обработки чата.
func (c *Config) GetCountChat() int {
	return c.getInt(configCountChat)
}
