package  v1beta1

import (
	"crypto/sha256"
	"io/ioutil"

        "github.com/ghodss/yaml"
        "github.com/golang/glog"
)

type UserCfg struct {
        MaxWallTime   int64    `yaml:"maxWallTime,omitempty"`
        MaxWallTimePod   int64    `yaml:"maxWallTime,omitempty"`
        DefaultQueue  string   `yaml:"defaultQueue"`
        AllowedQueues []string `yaml:"allowedQueues"`
}

type Conf struct {
        AllowedFileSystem []string        `yaml:"allowedFileSystem"`
        UserDefault       UserCfg            `yaml:"userDefault"`
        Users             map[string]UserCfg `yaml:"users"`
}

func (c Conf) GetUserCfg(name string) UserCfg {
        u, ok := c.Users[name]
        if ok {
                if u.MaxWallTimePod == 0 {
                        u.MaxWallTimePod = c.UserDefault.MaxWallTimePod
                }
                if u.MaxWallTime == 0 {
                        u.MaxWallTime = c.UserDefault.MaxWallTime
                }
                if u.DefaultQueue == "" {
                        u.DefaultQueue = c.UserDefault.DefaultQueue
                }
                if u.AllowedQueues == nil {
                        u.AllowedQueues = c.UserDefault.AllowedQueues
                }

        } else {
                u = c.UserDefault
        }
        return u
}

func (c *Conf)LoadConfig(configFile string) (error) {
        data, err := ioutil.ReadFile(configFile)
        if err != nil {
                return  err
        }
        glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

        if err := yaml.Unmarshal(data, c); err != nil {
                return  err
        }

        return  nil
}
