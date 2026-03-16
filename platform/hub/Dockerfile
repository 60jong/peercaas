# ─── Build Stage ──────────────────────────────────────────────────────────────
FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /workspace

COPY . .

RUN chmod +x gradlew && ./gradlew :hub:bootJar -x test

RUN find hub/build/libs -name "*.jar" ! -name "*plain*" \
      -exec cp {} /workspace/app.jar \;

# ─── Runtime Stage ────────────────────────────────────────────────────────────
FROM eclipse-temurin:21-jre-alpine

WORKDIR /app

COPY --from=builder /workspace/app.jar app.jar

ENTRYPOINT ["java", "-jar", "app.jar"]
