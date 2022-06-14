all: libemail.so

SRCS=main.go formats.go utils.go libemail.go

libemail.so: $(SRCS) *.gotmpl *.png
	go build -buildmode c-shared -o $@ $(SRCS)

clean:
	rm -f libemailgo.h libemailgo.so libemail.h libemail.so

