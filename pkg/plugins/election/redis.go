package election

import (
	"context"
	"fmt"
	"prometheus-deepflow-adapter/pkg/config"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
)

type redisElector struct {
	uuid     string
	config   *RedisConfig
	client   *redis.Client
	isLeader *atomic.Bool
}

func NewRedisElector(config config.Configuration) (Election, error) {
	conf := config.(*RedisConfig)
	return &redisElector{
		client: redis.NewClient(&redis.Options{
			Addr:     conf.Addr,
			Password: conf.Passwd,
		}),
		uuid:     uuid.NewString(),
		isLeader: &atomic.Bool{},
		config:   conf,
	}, nil
}

func (r *redisElector) ping(ctx context.Context) error {
	res := r.client.Ping(ctx)
	return res.Err()
}

func (r *redisElector) trySet(ctx context.Context) error {
	res := r.client.SetNX(ctx, r.config.Key, r.uuid, r.config.HeartBeat)
	return res.Err()
}

func (r *redisElector) StartLeading(ctx context.Context) error {
	if err := r.ping(ctx); err != nil {
		return err
	}

	if err := r.trySet(ctx); err != nil {
		return err
	}
	r.isLeader.Store(true)
	return nil
}

func (r *redisElector) Release(ctx context.Context) error {
	if r.IsLeader() {
		res := r.client.Del(ctx, r.config.Key)
		if res.Err() != nil {
			return res.Err()
		}
	}
	r.isLeader.Store(false)
	return nil
}

func (r *redisElector) IsLeader() bool {
	return r.isLeader.Load()
}

func (r *redisElector) RetryPeriod() time.Duration {
	return r.config.RetryPeriod
}

func (r *redisElector) HeartBeat() time.Duration {
	return r.config.HeartBeat
}

func (r *redisElector) KeepAlive(ctx context.Context) {
	r.client.Expire(ctx, r.config.Key, r.config.HeartBeat)
	r.isLeader.Store(true)
}

type RedisConfig struct {
	Addr        string        `mapstructure:"addr"`
	Passwd      string        `mapstructure:"passwd"`
	Key         string        `mapstructure:"key"`
	HeartBeat   time.Duration `mapstructure:"heartbeat"`
	RetryPeriod time.Duration `mapstructure:"retry-period"`
}

func NewRedisConfig() config.Configuration {
	return &RedisConfig{}
}

func (r *RedisConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("redis", pflag.ContinueOnError)
	fs.StringVar(&r.Addr, "addr", "127.0.0.1:6379", "redis address")
	fs.StringVar(&r.Passwd, "passwd", "", "redis password")
	fs.StringVar(&r.Key, "key", "p8s-df-adapter-lock", "redis lock leader key")
	fs.DurationVar(&r.HeartBeat, "heartbeat", 15*time.Second, "lock heartbeat interval")
	fs.DurationVar(&r.RetryPeriod, "retry-period", 10*time.Second, "lock retry interval")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "redis", f.Name)
	})
	return fs
}

func init() {
	config.RegisterConfig(string(config.Redis), NewRedisConfig)
	RegisterElector(config.Redis, NewRedisElector)
}
