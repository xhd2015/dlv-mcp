- PRD.md
it is possible to set a breakpoint at specific go function, when that breakpoint hits, we execute a go function, we pass the function name and args to the go handler, and checks if it returns true, if true, we return immediately. You can do a deep extensive online research, and put your research report under tracing/break_go/PRD.md including full list of reference links.

- dlv_trap.md
After evaluated all the possibilities, I find it might be more practical to base on delve's solution. Can you also do a deep research and output a report into tracing/break_go/dlv_trap.md with full list of reference links. The report should be extensive, including proof of concept.

- dlv_trap_source_code.md
the client-server implementation is kind of limited. Let's add another report that describes if it is possible to directly modify the delve's source code to support call-handler-on-breakpoint. You should do deep dive first, specifically research the go devle's official repository on github. Then output the report into  tracing/break_go/dlv_trap_source_code.md. The report should be extensive, including proof of concept, and include full list of reference links.

- dlv_trap_source_code_func_interception.md
Function Call Interception looks promising. Can you do a more deep dive into this ,then output the report into tracing/break_go/dlv_trap_source_code_func_interception.md? The delve source has already been downloaded by you at workspace/delve. You can research that repo and also do online search. The report should be extensive, including proof of concept, and include full list of reference links.

- go_test_cover.md
in the document @runtime_instrumentation.md , it mentions go test cover. Can you do a in-depth and extensive research on this? you can refer to official go repository at @https://github.com/golang/go (already downloaded at workspace/go), and other online materials including go.dev. You need to output a research report to tracing/break_go/go_test_cover.md, report should be extensive, including proof of concept, and include full list of reference links.