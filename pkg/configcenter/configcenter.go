// Package configcenter 提供基于 go-zero 官方配置中心的公共配置加载能力。
package configcenter

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	zeroconfigcenter "github.com/zeromicro/go-zero/core/configcenter"
	"github.com/zeromicro/go-zero/core/configcenter/subscriber"
	"sigs.k8s.io/yaml"
)

var (
	// ErrInvalidConfig 表示传入的配置缺少有效的 ConfigCenter 配置。
	ErrInvalidConfig = errors.New("invalid config center configuration")
	// ErrRemoteConfigNotFound 表示配置中心中没有找到指定的远程配置。
	ErrRemoteConfigNotFound = errors.New("remote config not found")
)

// EtcdConfig 复用 go-zero 官方的 etcd 配置结构。
type EtcdConfig = subscriber.EtcdConf

// ConfigCenter 是配置中心配置的根节点。
type ConfigCenter struct {
	Etcd EtcdConfig
}

// SubscriberFactory 创建 go-zero 官方配置中心订阅器。
type SubscriberFactory func(config EtcdConfig) (subscriber.Subscriber, error)

// Loader 负责从 ConfigCenter 指定的 etcd 节点读取远程配置，并覆盖本地配置。
type Loader struct {
	factory SubscriberFactory
}

// Option 配置 Loader 的创建选项。
type Option func(*Loader)

// WithSubscriberFactory 使用自定义订阅器工厂，便于测试或替换配置中心实现。
func WithSubscriberFactory(factory SubscriberFactory) Option {
	return func(loader *Loader) {
		if factory != nil {
			loader.factory = factory
		}
	}
}

// New 创建一个配置加载器。
func New(options ...Option) *Loader {
	loader := &Loader{
		factory: subscriber.NewEtcdSubscriber,
	}

	for _, option := range options {
		option(loader)
	}

	return loader
}

// Load 从传入配置的 ConfigCenter.Etcd 读取远程 YAML，并覆盖传入配置中已有的字段。
func Load(config any) error {
	return New().Load(config)
}

// Load 从传入配置的 ConfigCenter.Etcd 读取远程 YAML，并覆盖传入配置中已有的字段。
func (loader *Loader) Load(config any) error {
	if loader == nil || loader.factory == nil {
		return fmt.Errorf("%w: loader is nil", ErrInvalidConfig)
	}

	center, err := extractConfigCenter(config)
	if err != nil {
		return err
	}
	if err := center.Etcd.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	configSubscriber, err := loader.factory(center.Etcd)
	if err != nil {
		return fmt.Errorf("create config center subscriber: %w", err)
	}
	if configSubscriber == nil {
		return fmt.Errorf("%w: config center subscriber is nil", ErrInvalidConfig)
	}

	// 使用官方 Configurator 获取原始字符串，避免官方 YAML 完整配置校验阻止部分字段覆盖。
	configurator, err := zeroconfigcenter.NewConfigCenter[string](
		zeroconfigcenter.Config{Type: "yaml", Log: false}, configSubscriber)
	if err != nil {
		return fmt.Errorf("load remote config %q: %w", center.Etcd.Key, err)
	}

	remoteConfig, err := configurator.GetConfig()
	if err != nil {
		return fmt.Errorf("get remote config %q: %w", center.Etcd.Key, err)
	}
	if remoteConfig == "" {
		return fmt.Errorf("%w: %s", ErrRemoteConfigNotFound, center.Etcd.Key)
	}

	if err := yaml.Unmarshal([]byte(remoteConfig), config); err != nil {
		return fmt.Errorf("override config from %q: %w", center.Etcd.Key, err)
	}

	return nil
}

// extractConfigCenter 从配置对象中读取名为 ConfigCenter 的导出字段。
func extractConfigCenter(config any) (ConfigCenter, error) {
	if config == nil {
		return ConfigCenter{}, fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	value := reflect.ValueOf(config)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return ConfigCenter{}, fmt.Errorf("%w: config must be a non-nil pointer", ErrInvalidConfig)
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return ConfigCenter{}, fmt.Errorf("%w: config must point to a struct", ErrInvalidConfig)
	}

	field := value.FieldByName("ConfigCenter")
	if !field.IsValid() {
		return ConfigCenter{}, fmt.Errorf("%w: ConfigCenter field is missing", ErrInvalidConfig)
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ConfigCenter{}, fmt.Errorf("%w: ConfigCenter is nil", ErrInvalidConfig)
		}
		field = field.Elem()
	}
	if !field.CanInterface() {
		return ConfigCenter{}, fmt.Errorf("%w: ConfigCenter field is not exported", ErrInvalidConfig)
	}

	data, err := json.Marshal(field.Interface())
	if err != nil {
		return ConfigCenter{}, fmt.Errorf("%w: encode ConfigCenter: %v", ErrInvalidConfig, err)
	}

	var center ConfigCenter
	if err := json.Unmarshal(data, &center); err != nil {
		return ConfigCenter{}, fmt.Errorf("%w: decode ConfigCenter: %v", ErrInvalidConfig, err)
	}

	return center, nil
}
