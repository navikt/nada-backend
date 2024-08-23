FROM openjdk:11

ARG METABASE_VERSION

WORKDIR /usr/src/metabase
ADD https://downloads.metabase.com/enterprise/${METABASE_VERSION}/metabase.jar /usr/src/metabase/
EXPOSE 3000

CMD ["java", "-jar", "metabase.jar"]
