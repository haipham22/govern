package redisstream

import "fmt"

func streamKey(namespace, stream string) string {
	return fmt.Sprintf("%s:%s", namespace, stream)
}
