package client

import "fmt"

var InvalidClusterKey = fmt.Errorf("invalid cluster key")

var InvalidClusterKeyResponseCode int = 526

var authTokenUnAuthorizedCode = 401

var authTokenSessionExpiredCode = 419
