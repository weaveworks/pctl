package profile

func SetProfileMakeArtifacts(makeArtifacts MakeArtifactsFunc) {
	profilesArtifactsMaker = makeArtifacts
}
