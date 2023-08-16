# Geneos Inbound API

The `api` package abstracts the inbound API available through the XML-RPC and REST based Netprobe APIs. Both APIs offer different levels of functionality and not all features are available in both.

You can use the package to directly connect and send data to a Netprobe or you can build and use a `plugin` that imitates Geneos plugins in targetted features but these work at the level of Dataviews on one sampler and, because of the API requirements, cannot create Samplers directly.

This README will give examples of using the package in both forms.

