package install

func SetProfileMakeArtifacts(makeArtifacts MakeArtifactsFunc) {
	profilesArtifactsMaker = makeArtifacts
}
