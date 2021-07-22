package install

func SetProfileMakeArtifacts(makeArtifacts MakeArtifactsFunc) {
	generateArtifacts = makeArtifacts
}
