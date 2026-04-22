package auth

import (
	"fmt"
	"pkg/auth/blacklist"
	"pkg/auth/jwt"
	"pkg/configs"

	"global_models/global_cache"
)

// ProviderConfig конфигурация для создания Auth провайдера
type ProviderConfig struct {
	// EnableBlacklist - включить проверку blacklist
	EnableBlacklist bool
}

// Provider DI провайдер для auth пакета
type Provider struct {
	jwtCfg      *configs.JWTConfig
	cache       global_cache.Cache
	providerCfg ProviderConfig

	jwtManager *jwt.Manager
	blacklist  *blacklist.RedisBlacklist
	auth       *Auth
}

// NewProvider создает новый провайдер
func NewProvider(jwtCfg *configs.JWTConfig) *Provider {
	return &Provider{
		jwtCfg: jwtCfg,
	}
}

// WithCache устанавливает cache для blacklist
func (p *Provider) WithCache(cache global_cache.Cache) *Provider {
	p.cache = cache
	return p
}

// WithConfig устанавливает дополнительные настройки
func (p *Provider) WithConfig(cfg ProviderConfig) *Provider {
	p.providerCfg = cfg
	return p
}

// Build создает и возвращает готовый Auth сервис
func (p *Provider) Build() (*Auth, error) {
	// 1. Создаем JWT Manager
	jwtManager, err := jwt.NewManager(p.jwtCfg)
	if err != nil {
		return nil, fmt.Errorf("create jwt manager: %w", err)
	}
	p.jwtManager = jwtManager

	// 2. Создаем Blacklist (если включен)
	var blacklistService *blacklist.RedisBlacklist
	if p.providerCfg.EnableBlacklist {
		if p.cache == nil {
			return nil, fmt.Errorf("cache is required when blacklist is enabled")
		}
		blacklistService = blacklist.NewRedisBlacklist(p.cache)
		p.blacklist = blacklistService
	}

	// 3. Создаем Auth сервис
	p.auth = New(jwtManager, blacklistService, p.providerCfg.EnableBlacklist)

	return p.auth, nil
}

// GetJWTManager возвращает JWT менеджер
func (p *Provider) GetJWTManager() *jwt.Manager {
	return p.jwtManager
}
