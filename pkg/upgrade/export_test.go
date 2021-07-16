package upgrade

func SetCopier(c func(src, dest string) error) {
	copy = c
}
