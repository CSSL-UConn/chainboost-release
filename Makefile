DETERLAB=simulation/platform/deterlab_users
SIMUL=simulation/manage/simulation
CWD:=$(shell pwd)

all: build

create-builddirs:
	@mkdir -p build

build-deterlab:
	@echo "building users(executable) for oses/arch"
	make -C ${DETERLAB}
	@echo "Moving files to build folder"
	@mv ${DETERLAB}/users build/

build-simul:
	@echo "building simul(executable) for oses/arch"
	make -C ${SIMUL}
	@echo "Moving files to build folder"
	@mv ${SIMUL}/simul build/


build: clean create-builddirs copy-configs build-deterlab build-simul


copy-configs:
	@echo "Copying Excel Files and Configs"
	@cp -r ${SIMUL}/*.toml build/
	@cp -r ${SIMUL}/*.xlsx build/

clean:
	@rm -rf build

deploy: all
	$(eval ARCH:=$(shell ssh ${USER}@csi-lab-ssh.engr.uconn.edu uname -m))
	$(eval OS:=$(shell ssh ${USER}@csi-lab-ssh.engr.uconn.edu uname -s))
	$(eval ARCH:=$(shell echo ${ARCH} | sed s/x86_64/amd64/))
	$(eval OS:=$(shell echo ${OS} | awk '{print tolower($0)}'))
	echo ${OS}
	echo ${ARCH}
	rsync -avz build/*.xlsx ${USER}@csi-lab-ssh.engr.uconn.edu:~/remote
	rsync -avz build/*.toml ${USER}@csi-lab-ssh.engr.uconn.edu:~/remote
	rsync -avz build/simul/${OS}/${ARCH}/simul ${USER}@csi-lab-ssh.engr.uconn.edu:~/remote
	rsync -avz build/users/${OS}/${ARCH}/users ${USER}@csi-lab-ssh.engr.uconn.edu:~/remote


.PHONY: clean create-builddirs build