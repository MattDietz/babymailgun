package main

import (
	"errors"
	"github.com/cerberus98/babymailgun"
	"github.com/spf13/viper"
	"strings"
	"testing"
)

func TestUpdateStatusInvalidRecipient(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 0}
	err := errors.New("550 Invalid recipient fail@unitests.com")
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusFailed {
		t.Errorf("Expected update.Status =='%s', got '%s'", babymailgun.StatusFailed, update.Status)
	}

	if update.Reason != babymailgun.ReasonInvalidRecipient {
		t.Errorf("Expected update.Reason == '%s', got '%s'", babymailgun.ReasonInvalidRecipient, update.Reason)
	}

	if update.Tries != 0 {
		t.Errorf("Expected Tries to be 0, got %d", update.Tries)
	}
}

func TestUpdateStatusBlankRecipient(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 0}
	err := errors.New(InvalidRecipientError)
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusFailed {
		t.Errorf("Expected update.Status =='%s', got '%s'", babymailgun.StatusFailed, update.Status)
	}

	if update.Reason != babymailgun.ReasonInvalidRecipient {
		t.Errorf("Expected update.Reason == '%s', got '%s'", babymailgun.ReasonInvalidRecipient, update.Reason)
	}

	if update.Tries != 0 {
		t.Errorf("Expected Tries to be 0, got %d", update.Tries)
	}
}

func TestUpdateStatusInvalidCommand(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 0}
	err := errors.New(UnrecognizedCommandError)
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusIncomplete {
		t.Errorf("Expected update.Status == '%s', got '%s'", babymailgun.StatusIncomplete, update.Status)
	}

	if update.Reason != babymailgun.ReasonUnrecognizedCommand {
		t.Errorf("Expected update.Reason == '%s', got '%s'", babymailgun.ReasonUnrecognizedCommand, update.Reason)
	}

	if update.Tries != 1 {
		t.Errorf("Expected Tries to be 1, got %d", update.Tries)
	}
}

func TestUpdateStatusEOF(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 0}
	err := errors.New(EOFError)
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusIncomplete {
		t.Errorf("Expected update.Status == '%s', got '%s'", babymailgun.StatusIncomplete, update.Status)
	}

	if update.Reason != babymailgun.ReasonEOF {
		t.Errorf("Expected update.Status == '%s', got '%s'", babymailgun.ReasonEOF, update.Reason)
	}

	if update.Tries != 1 {
		t.Errorf("Expected Tries to be 1, got %d", update.Tries)
	}
}

func TestUpdateExhaustedTriesStatusFailed(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 2}
	err := errors.New(EOFError)
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusFailed {
		t.Errorf("Expected update.Status == '%s', got '%s'", babymailgun.StatusFailed, update.Status)
	}

	if update.Reason != babymailgun.ReasonEOF {
		t.Errorf("Expected update.Reason == '%s', got '%s'", babymailgun.ReasonEOF, update.Reason)
	}

	if update.Tries != 3 {
		t.Errorf("Expected Tries to be 3, got %d", update.Tries)
	}
}

func TestUpdateStatusComplete(t *testing.T) {
	update := babymailgun.EmailUpdate{Tries: 0}
	err := error(nil)
	config := WorkerConfig{SendRetries: 3}

	ErrorToUpdateStatus(err, &config, &update)
	if update.Status != babymailgun.StatusComplete {
		t.Errorf("Expected update.Status == '%s', got '%s'", babymailgun.StatusFailed, update.Status)
	}

	if update.Reason != "" {
		t.Errorf("Expected update.Reason == '' got '%s'", update.Reason)
	}

	if update.Tries != 0 {
		t.Errorf("Expected Tries to be 0, got %d", update.Tries)
	}
}

func TestLoadConfigMissingRequiredVars(t *testing.T) {
	_, err := loadConfig()
	if err == nil || (err != nil && !strings.HasSuffix(err.Error(), "DB_HOST, DB_PORT, DB_NAME")) {
		t.Errorf("Expected Error with missing config vars 'DB_HOST, DB_PORT, DB_NAME' instead got %s", err)
	}
}

func TestLoadConfigInvalidNegativeInteger(t *testing.T) {
	viper.Set("DB_HOST", "127.0.0.1")
	viper.Set("DB_PORT", "27017")
	viper.Set("DB_NAME", "testing")
	viper.Set("WORKER_SLEEP", -1)
	_, err := loadConfig()
	if err == nil || (err != nil && !strings.HasPrefix(err.Error(), "Config WORKER_SLEEP")) {
		t.Errorf("Expected Error for WORKER_SLEEP instead got %s", err)
	}
	viper.Reset()
}

func TestLoadConfigMHasVars(t *testing.T) {
	viper.Set("DB_HOST", "127.0.0.1")
	viper.Set("DB_PORT", "27017")
	viper.Set("DB_NAME", "testing")
	wc, err := loadConfig()
	if err != nil {
		t.Errorf("Expected no error, instead got '%s'", err.Error())
	}

	if wc.Host != "127.0.0.1" {
		t.Errorf("Expected wc.Host == '127.0.0.1'. instead got '%s'", wc.Host)
	}

	if wc.Port != "27017" {
		t.Errorf("Expected wc.Port == '27017'. instead got '%s'", wc.Port)
	}

	if wc.DatabaseName != "testing" {
		t.Errorf("Expected wc.DatabaseName == 'testing'. instead got '%s'", wc.DatabaseName)
	}

	if wc.WorkerSleep != 10 {
		t.Errorf("Expected wc.WorkerSleep == 10. instead got '%d'", wc.WorkerSleep)
	}

	if wc.WorkerPool != 5 {
		t.Errorf("Expected wc.WorkerPool == 5. instead got '%d'", wc.WorkerPool)
	}

	if wc.ConnectionRetries != 3 {
		t.Errorf("Expected wc.ConnectionRetries == 3. instead got '%d'", wc.ConnectionRetries)
	}

	if wc.ConnectionTimeout != 30 {
		t.Errorf("Expected wc.ConnectionTimeout == 30. instead got '%d'", wc.ConnectionTimeout)
	}

	if wc.SendRetries != 3 {
		t.Errorf("Expected wc.SendRetries == 3. instead got '%d'", wc.SendRetries)
	}

	if wc.SendRetryInterval != 600 {
		t.Errorf("Expected wc.SendRetryInterval == 600. instead got '%d'", wc.SendRetryInterval)
	}
	viper.Reset()
}
