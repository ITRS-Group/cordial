all: libalert.so

SRCS=main.go formats.go libalert.go

libalert.so: $(SRCS) *.gotmpl *.png
	go build --ldflags '-s -w' -buildmode c-shared -o $@ $(SRCS)

clean:
	rm -f libalert.h libalert.so

