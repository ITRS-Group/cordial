all: libemail.so

SRCS=main.go formats.go libemail.go

libemail.so: $(SRCS) *.gotmpl *.png
	go build -buildmode c-shared -o $@ $(SRCS)

clean:
	rm -f libemail.h libemail.so

