'use client';

import { useEffect, useState } from 'react';
import { Badge, Button, Card, EmptyState, ErrorState, Input, Modal, Skeleton, Textarea } from '@/components/ui';
import { api } from '@/lib/api';
import type { Channel, Template } from '@/types';

const ALL_CHANNELS: Channel[] = ['websocket', 'webpush', 'email', 'whatsapp', 'sms'];

export default function TemplatesPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

  function load() {
    setLoading(true);
    api<{ data: Template[] | null }>('/api/v1/templates')
      .then((res) => setTemplates(res.data ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setOpen(true)}>New template</Button>
      </div>
      {error && <ErrorState message={error} />}
      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      ) : templates.length === 0 ? (
        <EmptyState title="No templates yet" hint="Create a template to reuse message bodies with variables." />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {templates.map((t) => (
            <Card key={t.id}>
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-medium">{t.name}</p>
                  <p className="font-mono text-xs text-muted">{t.slug}</p>
                </div>
              </div>
              <div className="mt-3 flex flex-wrap gap-1">
                {t.channels.map((c) => (
                  <Badge key={c} color="primary">
                    {c}
                  </Badge>
                ))}
              </div>
              {t.variables.length > 0 && (
                <p className="mt-3 text-xs text-muted">Variables: {t.variables.join(', ')}</p>
              )}
            </Card>
          ))}
        </div>
      )}
      <CreateTemplateModal
        open={open}
        onClose={() => setOpen(false)}
        onCreated={() => {
          setOpen(false);
          load();
        }}
      />
    </div>
  );
}

function CreateTemplateModal({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('');
  const [channels, setChannels] = useState<Channel[]>(['email']);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function toggle(c: Channel) {
    setChannels((prev) => (prev.includes(c) ? prev.filter((x) => x !== c) : [...prev, c]));
  }

  async function submit() {
    setLoading(true);
    setError(null);
    try {
      await api('/api/v1/templates', {
        method: 'POST',
        body: {
          name,
          channels,
          subject,
          body_email: body,
          body_websocket: body,
          body_webpush: body,
          body_whatsapp: body,
          body_sms: body,
        },
      });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'create failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="New template">
      <div className="space-y-4">
        {error && <ErrorState message={error} />}
        <Input placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
        <div className="flex flex-wrap gap-2">
          {ALL_CHANNELS.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => toggle(c)}
              className={`rounded-md px-3 py-1 text-xs ${
                channels.includes(c) ? 'bg-primary text-white' : 'bg-white/5 text-muted'
              }`}
            >
              {c}
            </button>
          ))}
        </div>
        <Input placeholder="Subject (supports {{variables}})" value={subject} onChange={(e) => setSubject(e.target.value)} />
        <Textarea
          placeholder="Body (use {{name}}, {{order_id}} …)"
          rows={5}
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        <Button onClick={submit} loading={loading} disabled={!name} className="w-full">
          Create template
        </Button>
      </div>
    </Modal>
  );
}
