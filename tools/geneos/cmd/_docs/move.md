Move, including rename, instances.

As any existing legacy .rc file is never changed, this will migrate the instance from .rc to JSON. The instance is stopped and restarted after the instance is moved. It is an error to try to move an instance to one that already exists with the same name.

If the component support rebuilding a templated configuration then this is run after the move but before the restart. This allows SANs to be updated as expected.

Moving across hosts is fully supported.
