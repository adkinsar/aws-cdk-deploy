rootDir=$(pwd)
cd lambda
make build
cd "$rootDir"
cdk syth