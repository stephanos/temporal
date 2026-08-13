#pragma D option quiet

pid$target::main.auditStart:entry
{
	started = 1;
	printf("GOMADV3_AUDIT_START\n");
}

pid$target::*clock_gettime*:entry,
pid$target::*mach_absolute_time*:entry
/started/
{
	printf("GOMADV3_HOST_CLOCK %s`%s\n", probemod, probefunc);
}
