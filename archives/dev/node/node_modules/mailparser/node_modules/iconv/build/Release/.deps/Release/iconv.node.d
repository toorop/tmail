cmd_Release/iconv.node := ln -f "Release/obj.target/iconv.node" "Release/iconv.node" 2>/dev/null || (rm -rf "Release/iconv.node" && cp -af "Release/obj.target/iconv.node" "Release/iconv.node")
