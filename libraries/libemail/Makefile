all: libemail.so

SRCS=main.go formats.go utils.go libemail.go

libemail.so: $(SRCS) *.gotmpl *.png
	go build -tags netgo,osusergo --ldflags '-s -w' -buildmode=c-shared -o $@ $(SRCS)

clean:
	rm -f libemail.h libemail.so
