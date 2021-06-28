## Possible solutions:

## Git merge only
1. Just do a git merge. Deal with the annoying fact that git parses line by line, its not aware of anything, its just a line diff.

## Kubectl apply
1. For each resource compare and merge. Doesn't deal with changes to raw local helm charts. can't be clever about resource renaming (git can sort of)

## Do all the things approach
1. For all `.yaml` files that are kubernetes resources, do the kubectl apply approach
2. For all non kubernetes yaml files, do the git merge approach.
3. ???
4. Profit
