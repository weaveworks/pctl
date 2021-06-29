#!/bin/bash
# Script to use custom ssh keys for various git repositories
# Run without arguments to get usage info.
#
# How it works:
# When used with SSH, git sends the path to the repository in the SSH command.
#   @see: https://github.com/git/git/blob/e870325/connect.c#L1268
# We extract this info and search for a key with the name.
# Based on the source, this seems to be used format since v2.0 at least.
#   @see: https://github.com/git/git/commit/a2036d7

if [[ $# -eq 0 ]]; then
	echo "Usage"
	echo "Set script as GIT_SSH_COMMAND"
	echo "Add SSH keys for git repositories under ~/.ssh/git-keys/ folder."
	echo "File name format:"
	echo "  For the repository git@github.com:github/practice.git"
	echo "  Put the private key into the file github-practice"
	echo "  (Note: slash converted to dash in path, no extension)"
	echo ""
	echo "Uses ssh by default, use GIT_SSH_COMMAND_REALSSH envvar to override."
	echo "For debugging set log output in envvar GIT_SSH_COMMAND_DEBUGLOG."
	exit 1
fi

function debuglog() {
	[ ! -z "$GIT_SSH_COMMAND_DEBUGLOG" ] && (echo `date +%FT%T` "$@") >> $GIT_SSH_COMMAND_DEBUGLOG
	return 0
}

for CMD_BUF in "$@"; do :; done

debuglog "Value of cmd.buf is: '$CMD_BUF'"

# @source: https://superuser.com/a/1142939/277157
declare -a "array=($( echo "$CMD_BUF" | sed 's/[][`~!@#$%^&*():;<>.,?/\|{}=+-]/\\&/g' ))"
for CMD_PATH in "${array[@]}"; do :; done
CMD_PATH=$(echo "$CMD_PATH" | sed 's/\\//g')
IDENTITY=
if [[ $CMD_PATH == *.git ]] ;
then
	REPOKEY=$(echo "$CMD_PATH" | sed 's/\.git//g' | sed 's/\//-/g')
	KEYFILE=$(echo /tmp/git-keys/$REPOKEY)
	if [[ -f "$KEYFILE" ]]
	then
		debuglog "Key '$KEYFILE' exists"
		IDENTITY=$(echo "-i $KEYFILE")
	else
		debuglog "Key '$KEYFILE' is missing"
	fi
else
	debuglog "No repo name detected. Skipping"
fi

SSH=${GIT_SSH_COMMAND_REALSSH:-ssh}
set -- $SSH $IDENTITY "$@"
debuglog "Calling with '$@'"

"$@"
