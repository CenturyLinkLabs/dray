FROM scratch
MAINTAINER CenturyLink Labs <clt-labs-futuretech@centurylink.com>
EXPOSE 3000

COPY dray /

ENTRYPOINT ["/dray"]
