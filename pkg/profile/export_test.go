package profile

func SetProfileGetter(profileGetter ProfileGetter) {
	getProfileDefinition = profileGetter
}

func SetProfileMakeArtifacts(makeArtifacts MakeArtifactsFunc) {
	profilesArtifactsMaker = makeArtifacts
}
