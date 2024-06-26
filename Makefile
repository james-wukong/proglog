GOCMD = go
CFSSL = cfssl
SSLGEN = $(CFSSL) gencert
GOBUILD = $(GOCMD) build
GOMOD = $(GOCMD) mod
GOTEST = $(GOCMD) test
CONFIG_PATH = ${HOME}/.proglog/

# The .PHONY directive in a Makefile is used to declare phony targets. 
# A phony target is not a real file; rather, it is a label for a recipe 
# to be executed when you explicitly invoke it. The primary purpose of 
# declaring phony targets is to avoid conflicts with file names and to 
# make the Makefile more readable and robust.
.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

.PHONY: gencert
gencert:
	$(SSLGEN) \
		-initca test/ca-csr.json | cfssljson -bare ca
	
	$(SSLGEN) \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server

	$(SSLGEN) \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		test/client-csr.json | cfssljson -bare client
		
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: tidy
tidy:
	$(GOMOD) tidy && $(GOMOD) vendor

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=.  \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

.PHONY: test
test:
	$(GOCMD) test -race ./..