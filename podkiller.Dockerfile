FROM scratch
ADD bin/podkiller /podkiller
CMD ["/podkiller"]
