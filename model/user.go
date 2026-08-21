package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const UserNameMaxLength = 20

var userSortColumns = map[string]string{
	"id":            "id",
	"username":      "username",
	"quota":         "quota",
	"group":         "group",
	"created_at":    "created_at",
	"last_login_at": "last_login_at",
}

type UserSortOptions struct {
	SortBy    string
	SortOrder string
}

func NewUserSortOptions(sortBy string, sortOrder string) UserSortOptions {
	normalizedSortBy := strings.ToLower(strings.TrimSpace(sortBy))
	normalizedSortOrder := strings.ToLower(strings.TrimSpace(sortOrder))
	if _, ok := userSortColumns[normalizedSortBy]; !ok {
		normalizedSortBy = "id"
		normalizedSortOrder = "desc"
	} else if normalizedSortOrder != "asc" {
		normalizedSortOrder = "desc"
	}

	return UserSortOptions{
		SortBy:    normalizedSortBy,
		SortOrder: normalizedSortOrder,
	}
}

func (options UserSortOptions) Apply(query *gorm.DB) *gorm.DB {
	columnName, ok := userSortColumns[options.SortBy]
	if !ok {
		columnName = "id"
	}
	q := query.Order(clause.OrderByColumn{
		Column: clause.Column{Name: columnName},
		Desc:   options.SortOrder != "asc",
	})
	if columnName != "id" {
		q = q.Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   true,
		})
	}
	return q
}

func resolveUserSortOptions(sortOptions []UserSortOptions) UserSortOptions {
	if len(sortOptions) == 0 {
		return NewUserSortOptions("", "")
	}
	return sortOptions[0]
}

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id               int                        `json:"id"`
	Username         string                     `json:"username" gorm:"unique;index" validate:"max=20"`
	Password         string                     `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	OriginalPassword string                     `json:"original_password" gorm:"-:all"` // this field is only for Password change verification, don't save it to database!
	DisplayName      string                     `json:"display_name" gorm:"index" validate:"max=20"`
	Role             int                        `json:"role" gorm:"type:int;default:1"`   // admin, common
	Status           int                        `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email            string                     `json:"email" gorm:"index" validate:"max=50"`
	GitHubId         string                     `json:"github_id" gorm:"column:github_id;index"`
	DiscordId        string                     `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId           string                     `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId         string                     `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId       string                     `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode string                     `json:"verification_code" gorm:"-:all"`                         // this field is only for Email verification, don't save it to database!
	AccessToken      *string                    `json:"-" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota            int                        `json:"quota" gorm:"type:int;default:0"`
	UsedQuota        int                        `json:"used_quota" gorm:"type:int;default:0;column:used_quota"` // used quota
	RequestCount     int                        `json:"request_count" gorm:"type:int;default:0;"`               // request number
	Group            string                     `json:"group" gorm:"type:varchar(64);default:'default'"`
	AffCode          string                     `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount         int                        `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota         int                        `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`           // 邀请剩余额度
	AffHistoryQuota  int                        `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"` // 邀请历史额度
	InviterId        int                        `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	DeletedAt        gorm.DeletedAt             `gorm:"index"`
	LinuxDOId        string                     `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	Setting          string                     `json:"setting" gorm:"type:text;column:setting"`
	Remark           string                     `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	StripeCustomer   string                     `json:"stripe_customer" gorm:"type:varchar(64);column:stripe_customer;index"`
	// RateLimitTotal: per-user total requests per minute. 0 means follow group setting.
	RateLimitTotal int `json:"rate_limit_total" gorm:"type:int;default:0;column:rate_limit_total"`
	// RateLimitSuccess: per-user successful requests per minute. 0 means follow group setting.
	RateLimitSuccess int                        `json:"rate_limit_success" gorm:"type:int;default:0;column:rate_limit_success"`
	CreatedAt        int64                      `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	LastLoginAt      int64                      `json:"last_login_at" gorm:"default:0;column:last_login_at"`
	AuthVersion      int64                      `json:"-" gorm:"type:bigint;not null;default:1;column:auth_version"`
	AdminPermissions map[string]map[string]bool `json:"admin_permissions,omitempty" gorm:"-:all"`
}

func (user *User) ToBaseUser() *UserBase {
	cache := &UserBase{
		Id:               user.Id,
		Group:            user.Group,
		Quota:            user.Quota,
		Status:           user.Status,
		Role:             user.Role,
		Username:         user.Username,
		Setting:          user.Setting,
		Email:            user.Email,
		RateLimitTotal:   user.RateLimitTotal,
		RateLimitSuccess: user.RateLimitSuccess,
		AuthVersion:      user.AuthVersion,
		CacheSchema:      userCacheSchemaVersion,
	}
	return cache
}
