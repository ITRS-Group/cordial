all: libalert.so

SRCS=main.go formats.go libalert.go

# the module files are prerequisites as the library is built from the
# cordial module; changes to other packages in the module need `make clean`
libalert.so: $(SRCS) *.gotmpl *.png ../../go.mod ../../go.sum
	go build -trimpath -tags netgo,osusergo --ldflags '-s -w' -buildmode c-shared -o $@ $(SRCS)

clean:
	rm -f libalert.h libalert.so
