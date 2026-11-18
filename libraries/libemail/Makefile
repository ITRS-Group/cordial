all: libemail.so

SRCS=main.go formats.go utils.go libemail.go

# the module files are prerequisites as the library is built from the
# cordial module; changes to other packages in the module need `make clean`
libemail.so: $(SRCS) *.gotmpl *.png ../../go.mod ../../go.sum
	go build -trimpath -tags netgo,osusergo --ldflags '-s -w' -buildmode=c-shared -o $@ $(SRCS)

clean:
	rm -f libemail.h libemail.so
