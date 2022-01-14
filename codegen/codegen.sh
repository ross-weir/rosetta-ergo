#!/bin/bash

OS="$(uname)"
case "${OS}" in
    'Linux')
        OS='linux'
        SED_IFLAG=(-i'')
        ;;
    'Darwin')
        OS='macos'
        SED_IFLAG=(-i '')
        ;;
    *)
        echo "Operating system '${OS}' not supported."
        exit 1
        ;;
esac

ERGO_SPEC_VERSION=master
# TODO: invalid types are fixed
#curl -L https://raw.githubusercontent.com/ergoplatform/ergo/${ERGO_SPEC_VERSION}/src/main/resources/api/openapi.yaml -o openapi.yaml

GENERATOR_VERSION=v4.3.0
docker run --user "$(id -u):$(id -g)" --rm -v "${PWD}":/local \
  openapitools/openapi-generator-cli:${GENERATOR_VERSION} generate \
  -i /local/openapi.yaml \
  -g go \
  -t /local/codegen/templates \
  --additional-properties packageName=ergo \
  --skip-validate-spec \
  -o /local/client_tmp;

rm -f client_tmp/go.mod;
rm -f client_tmp/README.md;
rm -f client_tmp/go.mod;
rm -f client_tmp/go.sum;
rm -rf client_tmp/api;
rm -rf client_tmp/docs;
rm -f client_tmp/git_push.sh;
rm -f client_tmp/.travis.yml;
rm -f client_tmp/.gitignore;
rm -f client_tmp/.openapi-generator-ignore;
rm -rf client_tmp/.openapi-generator;
# TODO: invalid types are fixed
#rm -f openapi.yaml;

# remove existing types
rm -f ergo/types/*.go;

mv client_tmp/model_*.go ergo/types/;
for file in ergo/types/model_*.go; do
    mv "$file" "${file/model_/}"
done

#rm -rf client_tmp;

# remove types we don't need or that currently have issues/are unfinished
# there's still a lot of unused types
rm -f ergo/types/boxes_request_holder.go
rm -f ergo/types/merkle_proof.go
rm -f ergo/types/requests_holder.go
rm -f ergo/types/transaction_hints_bag.go
rm -f ergo/types/proof_of_upcoming_transactions.go
rm -f ergo/types/transaction_signing_request.go
rm -f ergo/types/inline_object*.go
rm -f ergo/types/inline_response*.go
rm -f ergo/types/work_message.go
rm -f ergo/types/scan*.go
rm -f ergo/types/*_predicate.go
rm -f ergo/types/sigma*.go
rm -f ergo/types/secret_proven.go
rm -f ergo/types/hint_extraction_request.go
rm -f ergo/types/ergo_like_context.go
rm -f ergo/types/crypto_result.go
rm -f ergo/types/commitment.go
rm -f ergo/types/commitment_with_secret.go
rm -f ergo/types/execute_script.go

# fix linting issues
sed "${SED_IFLAG[@]}" 's/Api/API/g' ergo/types/*;
sed "${SED_IFLAG[@]}" 's/Json/JSON/g' ergo/types/*;
sed "${SED_IFLAG[@]}" 's/Id /ID /g' ergo/types/*;
sed "${SED_IFLAG[@]}" 's/Url/URL/g' ergo/types/*;

# format generated code
FORMAT_GEN="gofmt -w /local/ergo/types"
GOLANG_VERSION=1.17
docker run --rm -v "${PWD}":/local \
  golang:${GOLANG_VERSION} sh -c \
  "cd /local; ${FORMAT_GEN}"
