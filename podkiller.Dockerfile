FROM scratch
ADD bin/podkiller /podkiller
ENTRYPOINT ["/podkiller"]
CMD ["-help"]
