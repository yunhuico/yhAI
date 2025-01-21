WORKDIR=$(shell pwd)
CMI_IMAGE="linkerrepository/linkerdcos_cmi"
cmi-build:
	cd $(WORKDIR)/cmi/resources/utilization-maximization && git pull
	cd $(WORKDIR) && docker build -t $(CMI_IMAGE) -f Dockerfile.cmi .

cmi-push: cmi-build
	docker push $(CMI_IMAGE)
