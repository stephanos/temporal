#pragma D option quiet

dtrace:::BEGIN
{
	printf("GOMAD3_AUDIT_READY\n");
}

pid$target::main.auditStart:entry
{
	started = 1;
	printf("GOMAD3_AUDIT_START\n");
}

pid$target::*clock_gettime*:entry,
pid$target::*mach_absolute_time*:entry
/started/
{
	printf("GOMAD3_HOST_CLOCK %s`%s\n", probemod, probefunc);
}
