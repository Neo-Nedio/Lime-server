package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		FindOneByEmailAndState(ctx context.Context, email string, status, deleted int64) (*User, error)
		CreateAuthUser(ctx context.Context, data *User) (sql.Result, error)
		UpdateAuthState(ctx context.Context, id, status, deleted int64) error
		withSession(session sqlx.Session) UserModel
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn),
	}
}

func (m *customUserModel) withSession(session sqlx.Session) UserModel {
	return NewUserModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customUserModel) FindOneByEmailAndState(ctx context.Context, email string, status, deleted int64) (*User, error) {
	var resp User
	query := fmt.Sprintf("select %s from %s where email = $1 and status = $2 and deleted = $3 limit 1", userRows, m.table)
	err := m.conn.QueryRowCtx(ctx, &resp, query, email, status, deleted)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *customUserModel) CreateAuthUser(ctx context.Context, data *User) (sql.Result, error) {
	normalizeAuthUser(data)
	return m.Insert(ctx, data)
}

func (m *customUserModel) UpdateAuthState(ctx context.Context, id, status, deleted int64) error {
	query := fmt.Sprintf("update %s set status = $1, deleted = $2, update_time = CURRENT_TIMESTAMP where id = $3", m.table)
	_, err := m.conn.ExecCtx(ctx, query, status, deleted, id)
	return err
}

func normalizeAuthUser(data *User) {
	data.Nickname = newNullString(data.Nickname.String)
	data.Handle = newNullString(data.Handle.String)
	data.Uid = newNullString(data.Uid.String)
	data.Bio = newNullString(data.Bio.String)
	data.Phone = newNullString(data.Phone.String)
	data.Avatar = newNullString(data.Avatar.String)
	data.BackgroundImage = newNullString(data.BackgroundImage.String)
	data.Region = newNullString(data.Region.String)
}

func newNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
