ARG BUILDER_IMAGE=fission/builder
FROM ${BUILDER_IMAGE} AS fission-builder


FROM microsoft/dotnet:2.0.0-sdk AS builderimage


WORKDIR /app

# Copy csproj and restore as distinct layers
COPY *.csproj ./
RUN dotnet restore

# Copy everything else and build
COPY . ./
RUN dotnet publish -c Release -o out


# Build runtime image
FROM microsoft/dotnet:aspnetcore-runtime
WORKDIR /app
COPY --from=builderimage /app/out .

# this builder is actually compilation from : https://github.com/fission/fission/tree/master/builder/cmd and renamed cmd.exe to builder
# make sure to compile it in linux only else you will get exec execute error as binary was compiled in windows and running on linux

COPY --from=fission-builder /builder /builder

# ADD builder /builder

ADD build.sh /usr/local/bin/build
RUN chmod +x /usr/local/bin/build

ADD build.sh /bin/build
RUN chmod +x /bin/build

EXPOSE 8001
