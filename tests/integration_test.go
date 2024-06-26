//go:build integration

package tests_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Hukyl/genesis-kma-school-entry/database"
	"github.com/Hukyl/genesis-kma-school-entry/mail"
	"github.com/Hukyl/genesis-kma-school-entry/mail/backends"
	mailCfg "github.com/Hukyl/genesis-kma-school-entry/mail/config"
	"github.com/Hukyl/genesis-kma-school-entry/models"
	"github.com/Hukyl/genesis-kma-school-entry/rate/fetchers"
	"github.com/Hukyl/genesis-kma-school-entry/server"
	serverCfg "github.com/Hukyl/genesis-kma-school-entry/server/config"
	"github.com/Hukyl/genesis-kma-school-entry/server/notifications"
	"github.com/Hukyl/genesis-kma-school-entry/server/notifications/message"
	"github.com/Hukyl/genesis-kma-school-entry/server/service"
	"github.com/Hukyl/genesis-kma-school-entry/settings"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ccFrom = "USD"
	ccTo   = "UAH"
)

func TestCurrencyBeaconFetchRate_NoAuthorization(t *testing.T) {
	// Arrange
	fetcher := fetchers.NewCurrencyBeaconFetcher("")
	// Act
	_, err := fetcher.FetchRate(context.Background(), ccFrom, ccTo)
	// Assert
	assert.Error(t, err)
}

func TestNBUFetchRate(t *testing.T) {
	// Arrange
	fetcher := fetchers.NewNBURateFetcher()
	// Act
	result, err := fetcher.FetchRate(context.Background(), ccFrom, ccTo)
	// Assert
	require.NoError(t, err)
	assert.Equal(t, ccFrom, result.CurrencyFrom)
	assert.Equal(t, ccTo, result.CurrencyTo)
	assert.NotZero(t, result.Rate)
}

func TestChainFetchRate_FailFirst(t *testing.T) {
	// Arrange
	nbuFetcher := fetchers.NewNBURateFetcher()
	curBeaconFetcher := fetchers.NewCurrencyBeaconFetcher("")
	curBeaconFetcher.SetNext(nbuFetcher)
	// Act
	result, err := curBeaconFetcher.FetchRate(context.Background(), ccFrom, ccTo)
	// Assert
	require.NoError(t, err)
	assert.NotZero(t, result.Rate)
}

func TestRateServiceFetchRate_Success(t *testing.T) {
	// Arrange
	rateRepo := models.NewRateRepository(database.SetUpTest(t, &models.Rate{}))
	nbuFetcher := fetchers.NewNBURateFetcher()
	rateFetcher := service.NewRateService(rateRepo, nbuFetcher)
	// Act
	result, err := rateFetcher.FetchRate(context.Background(), ccFrom, ccTo)
	// Assert
	require.NoError(t, err)
	assert.Equal(t, ccFrom, result.CurrencyFrom)
	assert.Equal(t, ccTo, result.CurrencyTo)
	assert.NotZero(t, result.Rate)
}

func TestUserNotificationsRecipients(t *testing.T) {
	// Arrange
	fromEmail := "example@gmail.com"
	ctx := context.Background()
	db := database.SetUpTest(t, &models.User{}, &models.Rate{})
	repo := models.NewUserRepository(db)

	rateService := service.NewRateService(
		models.NewRateRepository(db),
		fetchers.NewNBURateFetcher(),
	)
	smtpmockServer := mail.MockSMTPServer(t)
	emailClient := mail.NewClient(backends.NewGomailMailer(
		mailCfg.Config{
			FromEmail:    fromEmail,
			SMTPHost:     mail.Localhost,
			SMTPPort:     strconv.Itoa(smtpmockServer.PortNumber()),
			SMTPUser:     "user",
			SMTPPassword: "password",
		},
	))
	messageFormatter := message.PlainRate{}
	notifier := notifications.NewUsersNotifier(
		emailClient, rateService, repo, &messageFormatter,
	)
	users := []models.User{
		{Email: "test1@gmail.com"},
		{Email: "test2@gmail.com"},
	}
	for _, user := range users {
		if err := repo.Create(&user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
	}
	// Act
	notifier.Notify(ctx)
	// Assert
	messages := smtpmockServer.Messages()
	assert.Len(t, messages, len(users))
	for i, user := range users {
		msg := messages[i]
		assert.Len(t, msg.RcpttoRequestResponse(), 1)
		rcpt := msg.RcpttoRequestResponse()[0][0]
		assert.Contains(t, rcpt, user.Email)
		assert.Contains(t, msg.MailfromRequest(), fromEmail)
	}
}

func TestSubscribeUser_Success(t *testing.T) {
	// Arrange
	db := database.SetUpTest(t, &models.User{})
	user := &models.User{Email: "example@gmail.com"}
	repo := models.NewUserRepository(db)
	engine := server.NewEngine(server.Client{
		Config:   serverCfg.Config{Port: "8080"},
		UserRepo: repo,
	})
	// Act
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, server.SubscribePath, nil)
	req.PostForm = map[string][]string{
		"email": {user.Email},
	}
	engine.ServeHTTP(rr, req)
	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	exists, err := repo.Exists(user)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSubscribeUser_Conflict(t *testing.T) {
	user := &models.User{Email: "example@gmail.com"}
	// Arrange
	repo := models.NewUserRepository(database.SetUpTest(t, &models.User{}))
	err := repo.Create(user)
	require.NoError(t, err)
	engine := server.NewEngine(server.Client{
		Config:   serverCfg.Config{Port: "8080"},
		UserRepo: repo,
	})
	// Act
	req := httptest.NewRequest(http.MethodPost, server.SubscribePath, nil)
	req.PostForm = map[string][]string{
		"email": {user.Email},
	}
	rr := httptest.NewRecorder()
	engine.ServeHTTP(rr, req)
	// Assert
	assert.Equal(t, http.StatusConflict, rr.Code)
	exists, err := repo.Exists(user)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.ReleaseMode)
	_ = settings.InitSettings()
	m.Run()
}
