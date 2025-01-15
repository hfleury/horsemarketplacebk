package repositories

import (
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type UserRepo struct {
	psql *db.Database
	User *models.User
}
