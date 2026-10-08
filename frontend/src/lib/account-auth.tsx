'use client';

import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { AccountUser, getAccountInfo, logoutAccount } from './api';
import { AccountDialog } from '@/components/account-dialog';

type DialogMode = 'login' | 'register';

interface AccountAuthValue {
  user: AccountUser | null;
  loading: boolean;
  emailEnabled: boolean;
  setUser: (user: AccountUser | null) => void;
  openAccountDialog: (mode?: DialogMode) => void;
  logout: () => Promise<void>;
}

const AccountAuthContext = createContext<AccountAuthValue | null>(null);

export function AccountAuthProvider({ children, emailEnabled }: { children: React.ReactNode; emailEnabled: boolean }) {
  const [user, setUser] = useState<AccountUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [dialog, setDialog] = useState<{ open: boolean; mode: DialogMode }>({ open: false, mode: 'login' });

  useEffect(() => {
    getAccountInfo()
      .then((response) => setUser(response.code === 200 ? response.data : null))
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const refreshAccount = () => {
      getAccountInfo()
        .then((response) => setUser(response.code === 200 ? response.data : null))
        .catch(() => undefined);
    };
    const onVisible = () => {
      if (document.visibilityState === 'visible') refreshAccount();
    };
    window.addEventListener('focus', refreshAccount);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.removeEventListener('focus', refreshAccount);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, []);

  const openAccountDialog = useCallback((mode: DialogMode = 'login') => {
    setDialog({ open: true, mode });
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutAccount();
    } finally {
      setUser(null);
    }
  }, []);

  return (
    <AccountAuthContext.Provider value={{ user, loading, emailEnabled, setUser, openAccountDialog, logout }}>
      {children}
      <AccountDialog
        open={dialog.open}
        emailEnabled={emailEnabled}
        initialMode={dialog.mode}
        onClose={() => setDialog((value) => ({ ...value, open: false }))}
        onAuthenticated={(account) => {
          setUser(account);
          setDialog((value) => ({ ...value, open: false }));
        }}
      />
    </AccountAuthContext.Provider>
  );
}

export function useAccountAuth() {
  const value = useContext(AccountAuthContext);
  if (!value) throw new Error('useAccountAuth must be used within AccountAuthProvider');
  return value;
}
