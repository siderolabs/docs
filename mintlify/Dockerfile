FROM node:24.13.0-alpine

# Install Mintlify CLI globally
RUN npm install -g mint@latest

# Set working directory
WORKDIR /docs

# Expose the default Mintlify dev server port
EXPOSE 3000

# Default entrypoint to run Mintlify dev server
ENTRYPOINT ["mint"]
