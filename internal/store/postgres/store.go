package postgres

import "github.com/webitel/im-providers-service/internal/store"

type rootStore struct {
	gates    store.GateStore
	meta     store.MetaAppStore
	facebook store.FacebookStore
	whatsapp store.WhatsAppStore
}

func NewStore(
	gates store.GateStore,
	meta store.MetaAppStore,
	facebook store.FacebookStore,
	whatsapp store.WhatsAppStore,
) store.Store {
	return &rootStore{
		gates:    gates,
		meta:     meta,
		facebook: facebook,
		whatsapp: whatsapp,
	}
}

func (s *rootStore) Gates() store.GateStore        { return s.gates }
func (s *rootStore) Meta() store.MetaAppStore      { return s.meta }
func (s *rootStore) Facebook() store.FacebookStore { return s.facebook }
func (s *rootStore) WhatsApp() store.WhatsAppStore { return s.whatsapp }
