package traq

import "context"

type Service interface {
	UserExistsByName(ctx context.Context, traqID string) (bool, error)
}
