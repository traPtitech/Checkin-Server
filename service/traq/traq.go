package traq

import (
	"context"
	"os"
	"strings"

	traqclient "github.com/traPtitech/go-traq"
	"go.uber.org/zap"
)

type TraQService struct {
	logger      *zap.Logger
	client      *traqclient.APIClient
	accessToken string
}

func NewTraQService(logger *zap.Logger) *TraQService {
	cfg := traqclient.NewConfiguration()

	baseURL := strings.TrimSpace(os.Getenv("TRAQ_API_BASE_URL"))
	if baseURL != "" {
		cfg.Servers = traqclient.ServerConfigurations{
			{
				URL:         baseURL,
				Description: "custom",
			},
		}
	}

	return &TraQService{
		logger:      logger,
		client:      traqclient.NewAPIClient(cfg),
		accessToken: strings.TrimSpace(os.Getenv("TRAQ_ACCESS_TOKEN")),
	}
}

func (s *TraQService) UserExistsByName(ctx context.Context, traqID string) (bool, error) {
	if strings.TrimSpace(traqID) == "" {
		return false, nil
	}

	if s.accessToken == "" {
		return false, ErrTraQAccessTokenNotSet
	}

	authCtx := context.WithValue(ctx, traqclient.ContextAccessToken, s.accessToken)
	users, _, err := s.client.UserAPI.GetUsers(authCtx).Name(traqID).Execute()
	if err != nil {
		s.logger.Error("failed to fetch users from traQ API", zap.String("traq_id", traqID), zap.Error(err))
		return false, err
	}

	for _, user := range users {
		if user.Name == traqID {
			return true, nil
		}
	}

	return false, nil
}
