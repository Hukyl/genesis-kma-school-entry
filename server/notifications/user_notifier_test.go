package notifications_test

import (
	"context"
	"testing"

	"github.com/Hukyl/genesis-kma-school-entry/models"
	"github.com/Hukyl/genesis-kma-school-entry/rate"
	"github.com/Hukyl/genesis-kma-school-entry/server/notifications"
	"github.com/stretchr/testify/mock"
)

type mockRateFetcher struct {
	mock.Mock
}

func (m *mockRateFetcher) FetchRate(from, to string) (rate.Rate, error) {
	args := m.Called(from, to)
	return args.Get(0).(rate.Rate), args.Error(1)
}

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) FindAll() ([]models.User, error) {
	args := m.Called()
	return args.Get(0).([]models.User), args.Error(1)
}

type mockMessageFormatter struct {
	mock.Mock
}

func (m *mockMessageFormatter) SetRate(rate rate.Rate) {
	m.Called(rate)
}

func (m *mockMessageFormatter) Subject() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockMessageFormatter) String() string {
	args := m.Called()
	return args.String(0)
}

type mockEmailClient struct {
	mock.Mock
}

func (m *mockEmailClient) SendEmail(ctx context.Context, email, subject, message string) error {
	args := m.Called(ctx, email, subject, message)
	return args.Error(0)
}

func TestUserNotify(t *testing.T) {
	// Arrange
	ctx := context.Background()

	rateFetcher := new(mockRateFetcher)
	rateFetcher.On("FetchRate", "USD", "UAH").Return(rate.Rate{Rate: 27.5}, nil)

	userRepository := new(mockUserRepository)
	userRepository.On("FindAll").Return([]models.User{
		{Email: "example@gmail.com"},
		{Email: "example2@gmail.com"},
	}, nil)

	emailClient := new(mockEmailClient)
	emailClient.On(
		"SendEmail", ctx, "example@gmail.com", mock.Anything, mock.Anything,
	).Return(nil).Once()
	emailClient.On(
		"SendEmail", ctx, "example2@gmail.com", mock.Anything, mock.Anything,
	).Return(nil).Once()

	messageFormatter := new(mockMessageFormatter)
	messageFormatter.On("SetRate", rate.Rate{Rate: 27.5}).Return()
	messageFormatter.On("Subject").Return("USD-UAH exchange rate")
	messageFormatter.On("String").Return("1 USD = 27.5 UAH")

	notifier := notifications.NewUsersNotifier(
		emailClient,
		rateFetcher,
		userRepository,
		messageFormatter,
	)
	// Act
	notifier.Notify(ctx)
	// Assert
	rateFetcher.AssertExpectations(t)
	userRepository.AssertExpectations(t)
	emailClient.AssertExpectations(t)
}
