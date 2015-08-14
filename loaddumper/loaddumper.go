package loaddumper

type AccountDataLoadDumper interface {
	Load() (*AccountData, error)
	Dump(*AccountData) error
}
