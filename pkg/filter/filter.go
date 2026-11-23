package filter

import "github.com/herraristotle/rad/pkg/model"

type FilterHandler interface {
	DoFilter(req *model.Request) bool
}
