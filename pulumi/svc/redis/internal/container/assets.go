package container

import _ "embed"

//go:embed files/redis.conf
var redisConf string

//go:embed files/redis-standard.conf
var redisStandardConf string

//go:embed files/update-nodes.sh
var updateNodesScript string

//go:embed files/cluster-bootstrap.sh
var clusterBootstrapScript string
