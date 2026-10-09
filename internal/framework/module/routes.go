package module

import "net/http"

// RouteRegistry is the small HTTP contract exposed to modules. Modules do not
// depend on chi or the application's router implementation.
type RouteRegistry interface {
	Get(path string, handler http.HandlerFunc)
	Post(path string, handler http.HandlerFunc)
	Put(path string, handler http.HandlerFunc)
	Patch(path string, handler http.HandlerFunc)
	Delete(path string, handler http.HandlerFunc)
	With(middleware ...func(http.Handler) http.Handler) RouteRegistry
}

type RouteRegistrar interface {
	RegisterRoutes(RouteRegistry) error
}

// PublicRouteRegistrar is for carefully scoped routes that deliberately live
// outside /api/v1, such as the trusted external-application gateway. Modules
// still receive the same narrow RouteRegistry contract.
type PublicRouteRegistrar interface {
	RegisterPublicRoutes(RouteRegistry) error
}

func (r *Registry) RegisterRoutes(routes RouteRegistry) error {
	order, err := r.Resolve()
	if err != nil {
		return err
	}
	for _, item := range order {
		registrar, ok := item.(RouteRegistrar)
		if !ok {
			continue
		}
		if err := registrar.RegisterRoutes(routes); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) RegisterPublicRoutes(routes RouteRegistry) error {
	order, err := r.Resolve()
	if err != nil {
		return err
	}
	for _, item := range order {
		registrar, ok := item.(PublicRouteRegistrar)
		if !ok {
			continue
		}
		if err := registrar.RegisterPublicRoutes(routes); err != nil {
			return err
		}
	}
	return nil
}
