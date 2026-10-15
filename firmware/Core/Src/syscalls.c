/*
 * Stubs das chamadas de sistema da newlib. O firmware não usa arquivos nem
 * stdio (o log vai direto para a USART), mas vsnprintf puxa estas funções.
 */
#include <errno.h>
#include <sys/stat.h>

int _close(int fd)
{
    (void)fd;
    errno = EBADF;
    return -1;
}

int _lseek(int fd, int offset, int whence)
{
    (void)fd;
    (void)offset;
    (void)whence;
    return 0;
}

int _read(int fd, char *buf, int len)
{
    (void)fd;
    (void)buf;
    (void)len;
    return 0;
}

int _write(int fd, const char *buf, int len)
{
    (void)fd;
    (void)buf;
    return len;
}

int _fstat(int fd, struct stat *st)
{
    (void)fd;
    st->st_mode = S_IFCHR;
    return 0;
}

int _isatty(int fd)
{
    (void)fd;
    return 1;
}
