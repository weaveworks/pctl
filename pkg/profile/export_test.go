package profile

func SetProfileGetter(profileGetter ProfileGetter) {
	getProfileDefinition = profileGetter
}
