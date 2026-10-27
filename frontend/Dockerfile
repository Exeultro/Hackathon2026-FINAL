FROM node:20-alpine AS build
WORKDIR /src
COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
# Host node_modules (e.g. from Windows) must not leak into the image.
RUN rm -rf node_modules dist && npm ci --include=dev && npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
